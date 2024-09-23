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
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh_Hans_CN"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
	"github.com/go-redis/redis/v8"
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
	// 配置解析
	global.Viper = config.Init(&global.Config)

	// 日志
	global.Logger = logger.New(&logger.Config{
		Path:         global.Config.Logger.Path,
		Level:        global.Config.Logger.Level,
		ReportCaller: global.Config.Logger.ReportCaller,
	})

	// Redis
	global.Redis = redis.NewClient(&redis.Options{
		Addr:     global.Config.Redis.Addr,
		Password: global.Config.Redis.Password,
		DB:       global.Config.Redis.Db,
	})

	// Cache
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

	// Gorm
	if global.Config.Gorm.Type == config.TypeMysql {
		var dns string
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
		// SQLite
		dbPath := global.Config.Gorm.Dbpath
		dns := fmt.Sprintf("file:%s?mode=rw", dbPath)
		global.DB = orm.NewSqlite(&orm.SqliteConfig{
			Dns:           dns,
			MaxIdleConns:  global.Config.Gorm.MaxIdleConns,
			MaxOpenConns:  global.Config.Gorm.MaxOpenConns,
		})
	}
	DatabaseAutoUpdate()

	// Validator
	ApiInitValidator()

	// OSS
	global.Oss = &upload.Oss{
		AccessKeyId:     global.Config.Oss.AccessKeyId,
		AccessKeySecret: global.Config.Oss.AccessKeySecret,
		Host:            global.Config.Oss.Host,
		CallbackUrl:     global.Config.Oss.CallbackUrl,
		ExpireTime:      global.Config.Oss.ExpireTime,
		MaxByte:         global.Config.Oss.MaxByte,
	}

	// JWT
	// fmt.Println(global.Config.Jwt.PrivateKey)
	// global.Jwt = jwt.NewJwt(global.Config.Jwt.PrivateKey, global.Config.Jwt.ExpireDuration*time.Second)

	// Locker
	global.Lock = lock.NewLocal()

	// Gin
	http.ApiInit()

}

func ApiInitValidator() {
	validate := validator.New()
	enT := en.New()
	cn := zh_Hans_CN.New()
	uni := ut.New(enT, cn)
	trans, _ := uni.GetTranslator("cn")
	err := zh_translations.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		// 退出
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
	global.Validator.VTrans = trans

	global.Validator.ValidStruct = func(i interface{}) []string {
		err := global.Validator.Validate.Struct(i)
		errList := make([]string, 0, 10)
		if err != nil {
			if _, ok := err.(*validator.InvalidValidationError); ok {
				errList = append(errList, err.Error())
				return errList
			}
			for _, err2 := range err.(validator.ValidationErrors) {
				errList = append(errList, err2.Translate(global.Validator.VTrans))
			}
		}
		return errList
	}
	global.Validator.ValidVar = func(field interface{}, tag string) []string {
		err := global.Validator.Validate.Var(field, tag)
		fmt.Println(err)
		errList := make([]string, 0, 10)
		if err != nil {
			if _, ok := err.(*validator.InvalidValidationError); ok {
				errList = append(errList, err.Error())
				return errList
			}
			for _, err2 := range err.(validator.ValidationErrors) {
				errList = append(errList, err2.Translate(global.Validator.VTrans))
			}
		}
		return errList
	}

}

func DatabaseAutoUpdate() {
	version := 103

	db := global.DB

	if global.Config.Gorm.Type == config.TypeMysql {
		// 检查存不存在数据库，不存在则创建
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
			// 新链接
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
		// 查找最后一个version
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
	// 如果是初次则创建一个默认用户
	var vc int64
	global.DB.Model(&model.Version{}).Count(&vc)
	if vc == 1 {
		group := &model.Group{
			Name: "默认组",
			Type: model.GroupTypeDefault,
		}
		service.AllService.GroupService.Create(group)
		groupShare := &model.Group{
			Name: "共享组",
			Type: model.GroupTypeShare,
		}
		service.AllService.GroupService.Create(groupShare)
		// 是true
		is_admin := true
		admin := &model.User{
			Username: "admin",
			Nickname: "管理员",
			Status:   model.COMMON_STATUS_ENABLE,
			IsAdmin:  &is_admin,
			GroupId:  1,
		}
		admin.Password = service.AllService.UserService.EncryptPassword("admin")
		global.DB.Create(admin)
	}

}
