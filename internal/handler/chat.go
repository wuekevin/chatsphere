package handler

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/middleware"
	"github.com/wuekevin/chatsphere/internal/repository"
	"github.com/wuekevin/chatsphere/internal/service"
	"github.com/wuekevin/chatsphere/internal/ws"
)

// RegisterChat wires chat and WebSocket routes.
func RegisterChat(
	r *gin.Engine,
	chSvc *service.ChannelService,
	msgSvc *service.MessageService,
	userRepo repository.UserRepository,
	registry *ws.HubRegistry,
	cfg *config.Config,
) {
	r.GET("/ws/:channel", middleware.OptionalAuth(cfg.Server.SecretKey), wsHandler(chSvc, userRepo, registry, cfg))

	api := r.Group("/api")
	api.GET("/channels", listChannelsHandler(chSvc))
	api.GET("/channels/:slug", getChannelHandler(chSvc))
	api.GET("/channels/:slug/qrcode", channelQRCodeHandler(chSvc))
	api.GET("/channels/:slug/messages", getMessagesHandler(chSvc, msgSvc))
}

func wsHandler(chSvc *service.ChannelService, userRepo repository.UserRepository, registry *ws.HubRegistry, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("channel")

		channel, err := chSvc.GetBySlug(slug)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			}
			return
		}
		hub := registry.GetOrCreate(channel.Slug, channel.ID)

		// anonID: stable 8-char hex derived from client IP + server salt.
		ip := c.ClientIP()
		sum := sha256.Sum256([]byte(ip + cfg.Server.SecretKey))
		anonID := fmt.Sprintf("%x", sum[:4])

		claims := middleware.GetClaims(c)

		var (
			userID      *uint64
			displayName string
			country     string
			avatarColor string
			role        string
			reputation  int
			isAnon      = true
		)

		if claims != nil {
			user, err := userRepo.GetByID(claims.UserID)
			if err == nil {
				if user.IsBanned {
					c.JSON(http.StatusForbidden, gin.H{"error": "your account has been banned"})
					return
				}
				uid := user.ID
				userID = &uid
				displayName = user.Username
				country = user.Country
				avatarColor = user.AvatarColor
				reputation = user.Reputation
				role = string(user.Role)
				isAnon = false
				if country == "" {
					country = detectCountry(c, cfg)
				}
			}
		}

		if isAnon {
			if !channel.AllowAnonymous {
				c.JSON(http.StatusForbidden, gin.H{"error": "this channel requires a registered account"})
				return
			}
			displayName = anonName(anonID, country)
			avatarColor = randomColor(displayName)
			country = detectCountry(c, cfg)
		}

		ws.ServeWS(hub, channel, c.Writer, c.Request,
			userID, displayName, country, avatarColor, role, reputation, isAnon, anonID)
	}
}

func listChannelsHandler(chSvc *service.ChannelService) gin.HandlerFunc {
	return func(c *gin.Context) {
		channels, err := chSvc.ListActive()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list channels"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"channels": channels})
	}
}

func getChannelHandler(chSvc *service.ChannelService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		channel, err := chSvc.GetBySlug(slug)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"channel": channel})
	}
}

func channelQRCodeHandler(chSvc *service.ChannelService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		channel, err := chSvc.GetBySlug(slug)
		if err != nil || channel.TelegramURL == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		png, err := qrcode.Encode(channel.TelegramURL, qrcode.Medium, 256)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "qr failed"})
			return
		}
		c.Data(http.StatusOK, "image/png", png)
	}
}

func getMessagesHandler(chSvc *service.ChannelService, msgSvc *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		channel, err := chSvc.GetBySlug(slug)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch channel"})
			}
			return
		}

		messages, err := msgSvc.ListByChannel(channel.ID, 100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch messages"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"messages": messages})
	}
}

// anonName returns a stable display name from an 8-hex-char anonID.
// 20 adjectives × 20 nouns = 400 possible names.
// detectCountry reads the visitor's ISO-3166-1 alpha-2 country code from the
// first configured header that returns a 2-character value.
// Falls back to CF-IPCountry then X-Country, then Accept-Language as a hint.
func detectCountry(c *gin.Context, cfg *config.Config) string {
	hdrs := cfg.Server.CountryHeaders
	if len(hdrs) == 0 {
		hdrs = []string{"CF-IPCountry", "X-Country"}
	}
	for _, h := range hdrs {
		if v := c.GetHeader(h); len(v) == 2 {
			return strings.ToUpper(v)
		}
	}
	// Best-effort: extract region subtag from Accept-Language (e.g. "zh-CN" → "CN").
	al := c.GetHeader("Accept-Language")
	for _, seg := range strings.Split(al, ",") {
		seg = strings.TrimSpace(strings.SplitN(seg, ";", 2)[0])
		parts := strings.Split(seg, "-")
		if len(parts) >= 2 {
			cc := strings.ToUpper(parts[len(parts)-1])
			if len(cc) == 2 {
				return cc
			}
		}
	}
	return ""
}

func anonName(anonID, country string) string {
	n, _ := strconv.ParseUint(anonID, 16, 64)

	switch strings.ToUpper(country) {
	case "CN":
		adj := []string{"快乐", "温柔", "勇敢", "智慧", "活泼", "善良", "热情", "体贴",
			"开朗", "阳光", "淡定", "豁达", "乐观", "坚毅", "清醒", "从容",
			"灵动", "悠然", "洒脱", "率真"}
		ani := []string{"熊猫", "老虎", "狐狸", "雄鹰", "兔子", "蝴蝶", "海豚", "金鱼",
			"梅花鹿", "丹顶鹤", "猎豹", "雪豹", "孔雀", "鸿雁", "松鼠", "貂",
			"白鹭", "翠鸟", "水獭", "灵猫"}
		return adj[n%uint64(len(adj))] + "的" + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "TW", "HK", "MO":
		adj := []string{"快樂", "溫柔", "勇敢", "智慧", "活潑", "善良", "熱情", "體貼",
			"開朗", "陽光", "淡定", "豁達", "樂觀", "堅毅", "清醒", "從容",
			"靈動", "悠然", "瀟灑", "率真"}
		ani := []string{"熊貓", "老虎", "狐狸", "雄鷹", "兔子", "蝴蝶", "海豚", "金魚",
			"梅花鹿", "丹頂鶴", "獵豹", "雪豹", "孔雀", "鴻雁", "松鼠", "貂",
			"白鷺", "翠鳥", "水獺", "靈貓"}
		return adj[n%uint64(len(adj))] + "的" + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "JP":
		adj := []string{"元気な", "優しい", "勇敢な", "賢い", "明るい", "穏やかな", "快活な", "素直な",
			"のんびりな", "温かい", "颯爽な", "涼やかな", "華やかな", "清らかな", "たくましい", "しなやかな",
			"あどけない", "ほがらかな", "伸びやかな", "晴れやかな"}
		ani := []string{"キツネ", "クマ", "タヌキ", "ウサギ", "シカ", "サル", "ワシ", "フクロウ",
			"ネコ", "パンダ", "オオカミ", "イルカ", "ハクチョウ", "コウモリ", "サギ", "タカ",
			"リス", "カワウソ", "クジャク", "ウグイス"}
		return adj[n%uint64(len(adj))] + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "KR":
		adj := []string{"행복한", "용감한", "지혜로운", "활발한", "따뜻한", "밝은", "다정한", "씩씩한",
			"부드러운", "날쌘", "차분한", "상쾌한", "화사한", "맑은", "든든한", "유쾌한",
			"상냥한", "당찬", "생생한", "명랑한"}
		ani := []string{"여우", "곰", "사슴", "독수리", "토끼", "고래", "돌고래", "학",
			"호랑이", "표범", "늑대", "부엉이", "다람쥐", "수달", "공작", "백조",
			"제비", "원숭이", "판다", "너구리"}
		return adj[n%uint64(len(adj))] + " " + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "RU":
		adj := []string{"Быстрый", "Смелый", "Мудрый", "Добрый", "Яркий", "Тихий", "Дикий", "Острый",
			"Свободный", "Чистый", "Тёплый", "Мягкий", "Бодрый", "Ловкий", "Крепкий", "Верный",
			"Лёгкий", "Гордый", "Живой", "Звонкий"}
		ani := []string{"Лис", "Медведь", "Волк", "Орёл", "Заяц", "Сова", "Рысь", "Олень",
			"Журавль", "Соболь", "Пума", "Ибис", "Крот", "Мотылёк", "Выдра", "Лебедь",
			"Ястреб", "Бобёр", "Норка", "Кречет"}
		return adj[n%uint64(len(adj))] + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "DE", "AT":
		adj := []string{"Schnell", "Mutig", "Klug", "Sanft", "Hell", "Flink", "Wild", "Klar",
			"Stark", "Kühl", "Warm", "Rein", "Frei", "Wach", "Scharf", "Treu",
			"Leise", "Stolz", "Leicht", "Frisch"}
		ani := []string{"Fuchs", "Bär", "Wolf", "Adler", "Hase", "Eule", "Luchs", "Hirsch",
			"Falke", "Marder", "Puma", "Ibis", "Krähe", "Schwan", "Fischotter", "Storch",
			"Wiesel", "Dachs", "Specht", "Reiher"}
		return adj[n%uint64(len(adj))] + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "FR", "BE":
		adj := []string{"Vif", "Brave", "Sage", "Doux", "Clair", "Agile", "Libre", "Fort",
			"Calme", "Vrai", "Pur", "Chaud", "Fier", "Léger", "Vaste", "Alerte",
			"Sûr", "Leste", "Frais", "Noble"}
		ani := []string{"Renard", "Ours", "Loup", "Aigle", "Lièvre", "Hibou", "Lynx", "Cerf",
			"Faucon", "Marte", "Cigogne", "Loutre", "Puma", "Ibis", "Cygne", "Héron",
			"Belette", "Blaireau", "Pic", "Corbeau"}
		return adj[n%uint64(len(adj))] + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "ES", "MX", "CO", "CL":
		adj := []string{"Rápido", "Valiente", "Sabio", "Suave", "Vivo", "Libre", "Fuerte", "Claro",
			"Firme", "Cálido", "Puro", "Fiel", "Listo", "Ágil", "Sereno", "Veloz",
			"Noble", "Fiero", "Fresco", "Vasto"}
		ani := []string{"Zorro", "Oso", "Lobo", "Águila", "Liebre", "Búho", "Lince", "Ciervo",
			"Halcón", "Marta", "Cigüeña", "Nutria", "Puma", "Ibis", "Cisne", "Garza",
			"Comadreja", "Tejón", "Pájaro", "Cuervo"}
		return adj[n%uint64(len(adj))] + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "PT", "BR":
		adj := []string{"Rápido", "Bravo", "Sábio", "Suave", "Vivo", "Livre", "Forte", "Claro",
			"Firme", "Quente", "Puro", "Fiel", "Esperto", "Ágil", "Sereno", "Veloz",
			"Nobre", "Fero", "Fresco", "Vasto"}
		ani := []string{"Raposa", "Urso", "Lobo", "Águia", "Lebre", "Coruja", "Lince", "Veado",
			"Falcão", "Marta", "Cegonha", "Lontra", "Puma", "Ibis", "Cisne", "Garça",
			"Doninha", "Texugo", "Pica-pau", "Corvo"}
		return adj[n%uint64(len(adj))] + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "IT":
		adj := []string{"Rapido", "Coraggioso", "Saggio", "Mite", "Vivo", "Libero", "Forte", "Chiaro",
			"Calmo", "Caldo", "Puro", "Fedele", "Sveglio", "Agile", "Sereno", "Veloce",
			"Nobile", "Fiero", "Fresco", "Vasto"}
		ani := []string{"Volpe", "Orso", "Lupo", "Aquila", "Lepre", "Gufo", "Lince", "Cervo",
			"Falco", "Martora", "Cicogna", "Lontra", "Puma", "Ibis", "Cigno", "Airone",
			"Donnola", "Tasso", "Picchio", "Corvo"}
		return adj[n%uint64(len(adj))] + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "TH":
		adj := []string{"เร็ว", "กล้า", "ฉลาด", "อ่อนโยน", "สดใส", "อิสระ", "แข็งแกร่ง", "ใจดี",
			"สงบ", "อบอุ่น", "บริสุทธิ์", "ซื่อสัตย์", "คล่องแคล่ว", "ร่าเริง", "เปิดเผย", "รวดเร็ว",
			"สูงส่ง", "ภาคภูมิ", "สดชื่น", "กว้างขวาง"}
		ani := []string{"จิ้งจอก", "หมี", "หมาป่า", "นกอินทรี", "กระต่าย", "นกฮูก", "กวาง", "เสือ",
			"เหยี่ยว", "นากน้ำ", "นกกระสา", "สิงโต", "แมวป่า", "หงส์", "นกยูง", "ลิง",
			"กวางเรนเดียร์", "นกกระเรียน", "นกขุนทอง", "กระรอก"}
		return adj[n%uint64(len(adj))] + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "ID", "MY":
		adj := []string{"Cepat", "Berani", "Bijak", "Lembut", "Cerah", "Bebas", "Kuat", "Baik",
			"Tenang", "Hangat", "Murni", "Setia", "Cerdas", "Gesit", "Damai", "Ceria",
			"Mulia", "Gagah", "Segar", "Luas"}
		ani := []string{"Rubah", "Beruang", "Serigala", "Elang", "Kelinci", "Rusa", "Harimau", "Macan",
			"Rajawali", "Berang-berang", "Bangau", "Merak", "Kucing hutan", "Angsa", "Monyet", "Simpanse",
			"Trenggiling", "Binturong", "Musang", "Kancil"}
		return adj[n%uint64(len(adj))] + " " + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "VN":
		adj := []string{"Nhanh", "Dũng cảm", "Khôn ngoan", "Nhẹ nhàng", "Tươi sáng", "Tự do", "Mạnh mẽ", "Tốt bụng",
			"Bình yên", "Ấm áp", "Thuần khiết", "Trung thực", "Thông minh", "Nhanh nhẹn", "Thanh thản", "Vui vẻ",
			"Cao quý", "Hùng dũng", "Tươi mát", "Rộng lớn"}
		ani := []string{"Cáo", "Gấu", "Sói", "Đại bàng", "Thỏ", "Nai", "Hổ", "Báo",
			"Chim ưng", "Rái cá", "Cò", "Công", "Mèo rừng", "Thiên nga", "Khỉ", "Đười ươi",
			"Sơn dương", "Tê tê", "Trăn", "Cá heo"}
		return adj[n%uint64(len(adj))] + " " + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	case "SA", "AE", "EG":
		adj := []string{"سريع", "شجاع", "حكيم", "لطيف", "مشرق", "حر", "قوي", "طيب",
			"هادئ", "دافئ", "نقي", "أمين", "ذكي", "رشيق", "هانئ", "مرح",
			"نبيل", "فخور", "منعش", "واسع"}
		ani := []string{"ثعلب", "دب", "ذئب", "نسر", "أرنب", "غزال", "نمر", "فهد",
			"صقر", "بومة", "لقلق", "طاووس", "وشق", "بجعة", "قرد", "فيل",
			"جمل", "صقر شاهين", "بلبل", "دلفين"}
		return adj[n%uint64(len(adj))] + " " + ani[(n/uint64(len(adj)))%uint64(len(ani))]
	default:
		adjectives := []string{
			"Quick", "Calm", "Bold", "Wise", "Bright", "Swift", "Cool", "Kind",
			"Brave", "Sharp", "Wild", "Keen", "Fair", "Free", "Pure", "Warm",
			"Soft", "Vast", "Dark", "Stark",
		}
		nouns := []string{
			"Fox", "Bear", "Wolf", "Hawk", "Lynx", "Deer", "Crow", "Hare",
			"Owl", "Pike", "Kite", "Mink", "Puma", "Ibis", "Wren", "Vole",
			"Newt", "Moth", "Frog", "Swan",
		}
		return adjectives[n%uint64(len(adjectives))] + nouns[(n/uint64(len(adjectives)))%uint64(len(nouns))]
	}
}
