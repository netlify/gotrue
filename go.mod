module github.com/netlify/gotrue

go 1.15

replace github.com/DataDog/datadog-go => github.com/DataDog/datadog-go v4.2.0+incompatible

require (
	github.com/badoux/checkmail v1.2.1
	github.com/beevik/etree v1.1.0
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/didip/tollbooth/v5 v5.2.0
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/google/uuid v1.1.2 // indirect
	github.com/imdario/mergo v0.3.11
	github.com/joho/godotenv v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/netlify/mailme v1.1.1
	github.com/onrik/gorm-logrus v0.3.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/rs/cors v1.7.0
	github.com/russellhaering/gosaml2 v0.6.0
	github.com/russellhaering/goxmldsig v1.1.0
	github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	github.com/vcraescu/go-paginator/v2 v2.0.0
	golang.org/x/crypto v0.0.0-20201217014255-9d1352758620
	golang.org/x/oauth2 v0.0.0-20201208152858-08078c50e5b5
	gopkg.in/DataDog/dd-trace-go.v1 v1.27.1
	gorm.io/driver/mysql v1.0.3
	gorm.io/driver/postgres v1.0.5
	gorm.io/driver/sqlite v1.1.4
	gorm.io/driver/sqlserver v1.0.5
	gorm.io/gorm v1.20.8
)
