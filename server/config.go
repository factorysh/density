package server

type Config struct {
	Validators    map[string]map[string]interface{} `yaml:"validators"`
	Configurators map[string]map[string]interface{} `yaml:"configurators"`
	Listen        string                            `yaml:"listen"`
	DataDir       string                            `yaml:"data_dir"`
	CPU           int                               `yaml:"cpu"`
	RAM           int                               `yaml:"ram"`
}
