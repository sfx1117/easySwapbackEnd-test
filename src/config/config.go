package config

import (
	"github.com/ProjectsTask/EasySwapBase/evm/erc"
	logging "github.com/ProjectsTask/EasySwapBase/logger"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb"
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	Api            Api               `toml:"api" mapstructure:"api" json:"api"`
	ProjectCfg     *ProjectCfg       `toml:"project_cfg" mapstructure:"project_cfg" json:"project_cfg"`
	Log            logging.LogConf   `toml:"log" mapstructure:"log" json:"log"`
	DB             gdb.Config        `toml:"db" mapstructure:"db" json:"db"`
	Kv             *KvConfig         `toml:"kv" mapstructure:"kv" json:"kv"`
	Evm            *erc.NftErc       `toml:"evm" mapstructure:"evm" json:"evm"`
	MetadataParse  *MetadataParse    `toml:"metadata_parse" mapstructure:"metadata_parse" json:"metadata_parse"`
	ChainSupported []*ChainSupported `toml:"chain_supported" mapstructure:"chain_supported" json:"chain_supported"`
	//ImageCfg       *image.Config     `toml:"image_cfg" mapstructure:"image_cfg" json:"image_cfg"`
}

type Api struct {
	Port   string `toml:"port" mapstructure:"port" json:"port"`
	MaxNum int64  `toml:"max_num" mapstructure:"max_num" json:"max_num"`
}

type ProjectCfg struct {
	Name string `toml:"name" mapstructure:"name" json:"name"`
}

type KvConfig struct {
	Redis []*Redis `toml:"redis" mapstructure:"redis" json:"redis"`
}

type Redis struct {
	MasterName string `toml:"master_name" mapstructure:"master_name" json:"master_name"`
	Host       string `toml:"host" json:"host"`
	Type       string `toml:"type" json:"type"`
	Pass       string `toml:"pass" json:"pass"`
}

type MetadataParse struct {
	NameTags       []string `toml:"name_tags" mapstructure:"name_tags" json:"name_tags"`
	ImageTags      []string `toml:"image_tags" mapstructure:"image_tags" json:"image_tags"`
	AttributesTags []string `toml:"attributes_tags" mapstructure:"attributes_tags" json:"attributes_tags"`
	TraitNameTags  []string `toml:"trait_name_tags" mapstructure:"trait_name_tags" json:"trait_name_tags"`
	TraitValueTags []string `toml:"trait_value_tags" mapstructure:"trait_value_tags" json:"trait_value_tags"`
}

type ChainSupported struct {
	Name     string `toml:"name" mapstructure:"name" json:"name"`
	ChainId  int    `toml:"chain_id" mapstructure:"chain_id" json:"chain_id"`
	Endpoint string `toml:"endpoint" mapstructure:"endpoint" json:"endpoint"`
}

// 解析配置文件到Config对象
func UnmarshalConfig(configFilePath string) (*Config, error) {
	viper.SetConfigFile(configFilePath)
	viper.SetConfigType("toml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("CNFT")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	config, err := DefaultConfig()
	if err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}
	return config, nil
}

func DefaultConfig() (*Config, error) {
	return &Config{}, nil
}
