module github.com/lun-zhang/zlutils/v7

go 1.12

require (
	github.com/aws/aws-xray-sdk-go v1.0.0-rc.5.0.20180720202646-037b81b2bf76
	github.com/fvbock/endless v0.0.0-20170109170031-447134032cb6
	github.com/gin-gonic/gin v1.6.2
	github.com/go-playground/validator v9.29.1+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gosexy/to v0.0.0-20141221203644-c20e083e3123
	github.com/hashicorp/consul/api v1.1.0
	github.com/lestrrat/go-file-rotatelogs v0.0.0-20180223000712-d3151e2a480f
	github.com/lun-zhang/gorm v1.9.14-beta.1.14.0
	github.com/prometheus/client_golang v1.7.0
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7
	gopkg.in/redis.v5 v5.2.9
	gopkg.in/yaml.v2 v2.3.0
	zlutils v0.0.0
)

replace zlutils v0.0.0 => ./
