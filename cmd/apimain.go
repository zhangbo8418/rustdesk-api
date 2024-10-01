package main

import (
	"Gwen/config"
	"Gwen/global"
	"Gwen/http"
	"Gwen/lib/cache"
	"Gwen/lib/lock"
	"Gwen/lib/logger"
	"Gwen/lib/orm"
	"Gwen/lib/upload"
	"Gwen/model"
	"Gwen/service"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh_Hans_CN"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
	"github.com/go-redis/redis/v8"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	nethttp "net/http"
	"reflect"
)

// @title 管理系统API
// @version 1.0
// @description 接口
// @basePath /api
// @securityDefinitions.apikey token
// @in header
// @name api-token
// @securitydefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	//配置解析
	global.Viper = config.Init(&global.Config)

	//日志
	global.Logger = logger.New(&logger.Config{
		Path:         global.Config.Logger.Path,
		Level:        global.Config.Logger.Level,
		ReportCaller: global.Config.Logger.ReportCaller,
	})

	InitI18n()

	//redis
	global.Redis = redis.NewClient(&redis.Options{
		Addr:     global.Config.Redis.Addr,
		Password: global.Config.Redis.Password,
		DB:       global.Config.Redis.Db,
	})

	//cache
	if global.Config.Cache.Type == cache.TypeFile {
		fc := cache.NewFileCache()
		fc.SetDir(global.Config.Cache.FileDir)
		global.Cache = fc
	} else if global.Config.Cache.Type == cache.TypeRedis {
		global.Cache = cache.NewRedis(&redis.Options{
			Addr:     global.Config.Cache.RedisAddr,
			Password: global.Config.Cache.RedisPwd,
			DB:       global.Config.Cache.RedisDb,
		})
	}

	//gorm
	var dns string
	if global.Config.Gorm.Type == config.TypeMysql {
		if global.Config.Mysql.Socket != "" {
			// 使用 Unix Socket 构建 DSN
			dns = fmt.Sprintf("%s:%s@unix(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				global.Config.Mysql.Username,
				global.Config.Mysql.Password,
				global.Config.Mysql.Socket,
				global.Config.Mysql.Dbname)
		} else {
			// 使用 TCP 构建 DSN
			dns = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				global.Config.Mysql.Username,
				global.Config.Mysql.Password,
				global.Config.Mysql.Addr,
				global.Config.Mysql.Dbname)
		}
		global.DB = orm.NewMysql(&orm.MysqlConfig{
			Dns:          dns,
			MaxIdleConns: global.Config.Gorm.MaxIdleConns,
			MaxOpenConns: global.Config.Gorm.MaxOpenConns,
		})
	} else {
		//sqlite
		dbPath := global.Config.Gorm.Dbpath
		dns = fmt.Sprintf("file:%s?mode=rw", dbPath)
		global.DB = orm.NewSqlite(&orm.SqliteConfig{
			Path:          dns,
			MaxIdleConns: global.Config.Gorm.MaxIdleConns,
			MaxOpenConns: global.Config.Gorm.MaxOpenConns,
		})
	}
	DatabaseAutoUpdate()

	//validator
	ApiInitValidator()

	//oss
	global.Oss = &upload.Oss{
		AccessKeyId:     global.Config.Oss.AccessKeyId,
		AccessKeySecret: global.Config.Oss.AccessKeySecret,
		Host:            global.Config.Oss.Host,
		CallbackUrl:     global.Config.Oss.CallbackUrl,
		ExpireTime:      global.Config.Oss.ExpireTime,
		MaxByte:         global.Config.Oss.MaxByte,
	}

	//jwt
	//fmt.Println(global.Config.Jwt.PrivateKey)
	//global.Jwt = jwt.NewJwt(global.Config.Jwt.PrivateKey, global.Config.Jwt.ExpireDuration*time.Second)

	//locker
	global.Lock = lock.NewLocal()

	//gin
	http.ApiInit()

}

func ApiInitValidator() {
	validate := validator.New()

	// 定义不同的语言翻译
	enT := en.New()
	cn := zh_Hans_CN.New()

	uni := ut.New(enT, cn)

	enTrans, _ := uni.GetTranslator("en")
	zhTrans, _ := uni.GetTranslator("zh_Hans_CN")

	err := zh_translations.RegisterDefaultTranslations(validate, zhTrans)
	if err != nil {
		panic(err)
	}
	err = en_translations.RegisterDefaultTranslations(validate, enTrans)
	if err != nil {
		panic(err)
	}

	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		label := field.Tag.Get("label")
		if label == "" {
			return field.Name
		}
		return label
	})
	global.Validator.Validate = validate
	global.Validator.UT = uni // 存储 Universal Translator
	global.Validator.VTrans = zhTrans

	global.Validator.ValidStruct = func(ctx *gin.Context, i interface{}) []string {
		err := global.Validator.Validate.Struct(i)
		lang := ctx.GetHeader("Accept-Language")
		if lang == "" {
			lang = global.Config.Lang
		}
		trans := getTranslatorForLang(lang)
		errList := make([]string, 0, 10)
		if err != nil {
			if _, ok := err.(*validator.InvalidValidationError); ok {
				errList = append(errList, err.Error())
				return errList
			}
			for _, err2 := range err.(validator.ValidationErrors) {
				errList = append(errList, err2.Translate(trans))
			}
		}
		return errList
	}
	global.Validator.ValidVar = func(ctx *gin.Context, field interface{}, tag string) []string {
		err := global.Validator.Validate.Var(field, tag)
		lang := ctx.GetHeader("Accept-Language")
		if lang == "" {
			lang = global.Config.Lang
		}
		trans := getTranslatorForLang(lang)
		errList := make([]string, 0, 10)
		if err != nil {
			if _, ok := err.(*validator.InvalidValidationError); ok {
				errList = append(errList, err.Error())
				return errList
			}
			for _, err2 := range err.(validator.ValidationErrors) {
				errList = append(errList, err2.Translate(trans))
			}
		}
		return errList
	}

}
func getTranslatorForLang(lang string) ut.Translator {
	switch lang {
	case "zh_CN":
		fallthrough
	case "zh-CN":
		fallthrough
	case "zh":
		trans, _ := global.Validator.UT.GetTranslator("zh_Hans_CN")
		return trans
	case "en":
		fallthrough
	default:
		trans, _ := global.Validator.UT.GetTranslator("en")
		return trans
	}
}
func DatabaseAutoUpdate() {
	version := 212

	db := global.DB

	if global.Config.Gorm.Type == config.TypeMysql {
		//检查存不存在数据库，不存在则创建
		dbName := db.Migrator().CurrentDatabase()
		fmt.Println("dbName", dbName)
		if dbName == "" {
			dbName = global.Config.Mysql.Dbname
			var dsnWithoutDB string
			if global.Config.Mysql.Socket != "" {
				// 使用 Unix Socket 构建 DSN
				dsnWithoutDB = fmt.Sprintf("%s:%s@unix(%s)/?charset=utf8mb4&parseTime=True&loc=Local",
					global.Config.Mysql.Username,
					global.Config.Mysql.Password,
					global.Config.Mysql.Socket)
			} else {
				// 使用 TCP 构建 DSN
				dsnWithoutDB = fmt.Sprintf("%s:%s@tcp(%s)/?charset=utf8mb4&parseTime=True&loc=Local",
					global.Config.Mysql.Username,
					global.Config.Mysql.Password,
					global.Config.Mysql.Addr)
			}
			dbWithoutDB := orm.NewMysql(&orm.MysqlConfig{
				Dns: dsnWithoutDB,
			})
			// 获取底层的 *sql.DB 对象，并确保在程序退出时关闭连接
			sqlDBWithoutDB, err := dbWithoutDB.DB()
			if err != nil {
				fmt.Printf("获取底层 *sql.DB 对象失败: %v\n", err)
				return
			}
			defer func() {
				if err := sqlDBWithoutDB.Close(); err != nil {
					fmt.Printf("关闭连接失败: %v\n", err)
				}
			}()

			err = dbWithoutDB.Exec("CREATE DATABASE IF NOT EXISTS " + dbName + " DEFAULT CHARSET utf8mb4").Error
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}

	if !db.Migrator().HasTable(&model.Version{}) {
		Migrate(uint(version))
	} else {
		//查找最后一个version
		var v model.Version
		db.Last(&v)
		if v.Version < uint(version) {
			Migrate(uint(version))
		}
	}

}

func Migrate(version uint) {
	fmt.Println("migrating....", version)
	err := global.DB.AutoMigrate(
		&model.Version{},
		&model.User{},
		&model.UserToken{},
		&model.Tag{},
		&model.AddressBook{},
		&model.Peer{},
		&model.Group{},
		&model.UserThird{},
		&model.Oauth{},
		&model.LoginLog{},
	)
	if err != nil {
		fmt.Println("migrate err :=>", err)
	}
	global.DB.Create(&model.Version{Version: version})
	//如果是初次则创建一个默认用户
	var vc int64
	global.DB.Model(&model.Version{}).Count(&vc)
	if vc == 1 {
		localizer := global.Localizer(&gin.Context{
			Request: &nethttp.Request{},
		})
		defaultGroup, _ := localizer.LocalizeMessage(&i18n.Message{
			ID: "DefaultGroup",
		})
		group := &model.Group{
			Name: defaultGroup,
			Type: model.GroupTypeDefault,
		}
		service.AllService.GroupService.Create(group)

		shareGroup, _ := localizer.LocalizeMessage(&i18n.Message{
			ID: "ShareGroup",
		})
		groupShare := &model.Group{
			Name: shareGroup,
			Type: model.GroupTypeShare,
		}
		service.AllService.GroupService.Create(groupShare)
		//是true
		is_admin := true
		admin := &model.User{
			Username: "admin",
			Nickname: "Admin",
			Status:   model.COMMON_STATUS_ENABLE,
			IsAdmin:  &is_admin,
			GroupId:  1,
		}
		admin.Password = service.AllService.UserService.EncryptPassword("admin")
		global.DB.Create(admin)
	}

}

func InitI18n() {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	bundle.LoadMessageFile(global.Config.Gin.ResourcesPath + "/i18n/en.toml")
	bundle.LoadMessageFile(global.Config.Gin.ResourcesPath + "/i18n/zh_CN.toml")
	global.Localizer = func(ctx *gin.Context) *i18n.Localizer {
		lang := ctx.GetHeader("Accept-Language")
		if lang == "" {
			lang = global.Config.Lang
		}
		if lang == "en" {
			return i18n.NewLocalizer(bundle, "en")
		} else {
			return i18n.NewLocalizer(bundle, lang, "en")
		}
	}

	//personUnreadEmails := localizer.MustLocalize(&i18n.LocalizeConfig{
	//	DefaultMessage: &i18n.Message{
	//		ID: "PersonUnreadEmails",
	//	},
	//	PluralCount: 6,
	//	TemplateData: map[string]interface{}{
	//		"Name":        "LE",
	//		"PluralCount": 6,
	//	},
	//})
	//personUnreadEmails, err := global.Localizer.LocalizeMessage(&i18n.Message{
	//	ID: "ParamsError",
	//})
	//fmt.Println(err, personUnreadEmails)

}
