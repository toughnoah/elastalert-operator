module github.com/toughnoah/elastalert-operator

go 1.16

replace github.com/bouk/monkey v1.0.2 => bou.ke/monkey v1.0.0

require (
	github.com/bouk/monkey v1.0.2
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/stretchr/testify v1.7.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	sigs.k8s.io/controller-runtime v0.9.5
)
