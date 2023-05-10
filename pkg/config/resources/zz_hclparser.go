package resources

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/shipyard-run/hclconfig"
)

// setupHCLConfig configures the HCLConfig package and registers the custom types
func SetupHCLConfig(callback hclconfig.ProcessCallback, variables map[string]string, variablesFiles []string) *hclconfig.Parser {
	cfg := hclconfig.DefaultOptions()
	cfg.ParseCallback = callback
	cfg.Variables = variables
	cfg.VariablesFiles = variablesFiles

	p := hclconfig.NewParser(cfg)

	// Register the types
	p.RegisterType(TypeBlueprint, &Blueprint{})
	p.RegisterType(TypeCertificateCA, &CertificateCA{})
	p.RegisterType(TypeCertificateLeaf, &CertificateLeaf{})
	p.RegisterType(TypeContainer, &Container{})
	p.RegisterType(TypeCopy, &Copy{})
	p.RegisterType(TypeDocs, &Docs{})
	p.RegisterType(TypeRemoteExec, &RemoteExec{})
	p.RegisterType(TypeHelm, &Helm{})
	p.RegisterType(TypeImageCache, &ImageCache{})
	p.RegisterType(TypeIngress, &Ingress{})
	p.RegisterType(TypeK8sCluster, &K8sCluster{})
	p.RegisterType(TypeK8sConfig, &K8sConfig{})
	p.RegisterType(TypeLocalExec, &LocalExec{})
	p.RegisterType(TypeNetwork, &Network{})
	p.RegisterType(TypeNomadCluster, &NomadCluster{})
	p.RegisterType(TypeNomadJob, &NomadJob{})
	p.RegisterType(TypeRandomNumber, &RandomNumber{})
	p.RegisterType(TypeSidecar, &Sidecar{})
	p.RegisterType(TypeTemplate, &Template{})

	// Register the custom functions
	p.RegisterFunction("jumppad", customHCLFuncJumppad)
	p.RegisterFunction("docker_ip", customHCLFuncDockerIP)
	p.RegisterFunction("docker_host", customHCLFuncDockerHost)
	p.RegisterFunction("data", customHCLFuncDataFolder)
	p.RegisterFunction("data_with_permissions", customHCLFuncDataFolderWithPermissions)
	p.RegisterFunction("random_number", customHCLFuncRandomNumber)
	p.RegisterFunction("random_creature", customHCLFuncRandomCreature)

	return p
}

func customHCLFuncJumppad() (string, error) {
	return utils.JumppadHome(), nil
}

// returns the docker host ip address
func customHCLFuncDockerIP() (string, error) {
	return utils.GetDockerIP(), nil
}

func customHCLFuncDockerHost() (string, error) {
	return utils.GetDockerHost(), nil
}

func customHCLFuncDataFolderWithPermissions(name string, permissions int) (string, error) {
	perms := os.FileMode(permissions)
	return utils.GetDataFolder(name, perms), nil
}

func customHCLFuncDataFolder(name string) (string, error) {
	perms := os.FileMode(0775)
	return utils.GetDataFolder(name, perms), nil
}

func customHCLFuncRandomCreature() (string, error) {
	ci := rand.Intn(99)
	ai := rand.Intn(99)

	return fmt.Sprintf("%s-%s", adjectives[ai], creatures[ci]), nil
}

func customHCLFuncRandomNumber(min int, max int) (int, error) {
	rn := rand.Intn(max-min) + min

	return rn, nil
}

var creatures = []string{
	"Balrog",
	"Ent",
	"Dragon",
	"Warg",
	"Nazgul",
	"Giant",
	"Troll",
	"Orc",
	"Uruk-hai",
	"Goblin",
	"Hobgoblin",
	"Dwarf",
	"Elf",
	"Gollum",
	"Spider",
	"Eagle",
	"Smaug",
	"Drake",
	"Rider",
	"Draugr",
	"Hobbit",
	"Wight",
	"Baluchitherium",
	"Harpy",
	"Minotaur",
	"Cyclops",
	"Satyr",
	"Manticore",
	"Chimera",
	"Cerberus",
	"Griffin",
	"Phoenix",
	"Wyvern",
	"Kraken",
	"Mermaid",
	"Selkie",
	"Kelpie",
	"Basilisk",
	"Medusa",
	"Sphinx",
	"Yeti",
	"Bigfoot",
	"Chupacabra",
	"Mothman",
	"Thunderbird",
	"Jotun",
	"Frost-Giant",
	"Fire-Giant",
	"Dullahan",
	"Leprechaun",
	"Unicorn",
	"Pegasus",
	"Centaur",
	"Naga",
	"Gorgon",
	"Vampire",
	"Werewolf",
	"Zombie",
	"Mummy",
	"Frankenstein",
	"Cthulhu",
	"Shoggoth",
	"Elder-Thing",
	"Deep-One",
	"Mi-go",
	"Yith",
	"Nyarlarthotep",
	"Shub-Niggurath",
	"Azathoth",
	"Boggart",
	"Pixie",
	"Nixie",
	"Sprite",
	"Gnome",
	"Brownie",
	"Imp",
	"Demon",
	"Incubus",
	"Succubus",
	"Devil",
	"Angel",
	"Seraph",
	"Cherub",
	"Harpy",
	"Cockatrice",
	"Banshee",
	"Ghoul",
	"Mimic",
	"Gargoyle",
	"Ogre",
	"Cave-Dweller",
	"Skeleton",
	"Giant-Rat",
	"Varghulf",
	"Vampire-Bat",
	"Giant-Scorpion",
	"Giant-Snake",
	"Giant-Crab",
	"Giant-Spider",
	"Great-Worm",
}

var adjectives = []string{
	"Sparkling",
	"Zesty",
	"Vibrant",
	"Wistful",
	"Glittering",
	"Mellifluous",
	"Rambunctious",
	"Fanciful",
	"Fluffy",
	"Whimsical",
	"Vivacious",
	"Ethereal",
	"Mysterious",
	"Radiant",
	"Lively",
	"Jubilant",
	"Cozy",
	"Majestic",
	"Iridescent",
	"Serene",
	"Enigmatic",
	"Intriguing",
	"Luxuriant",
	"Mystic",
	"Delightful",
	"Dreamy",
	"Enchanting",
	"Mesmerizing",
	"Playful",
	"Quirky",
	"Luminous",
	"Effervescent",
	"Sensational",
	"Colorful",
	"Charming",
	"Elegant",
	"Blissful",
	"Jolly",
	"Silly",
	"Mischievous",
	"Groovy",
	"Pensive",
	"Soulful",
	"Breezy",
	"Witty",
	"Mystical",
	"Graceful",
	"Whirlwind",
	"Rustic",
	"Impressive",
	"Radiant",
	"Spontaneous",
	"Eccentric",
	"Bohemian",
	"Blissful",
	"Playful",
	"Clever",
	"Captivating",
	"Fascinating",
	"Haunting",
	"Harmonious",
	"Joyful",
	"Melancholic",
	"Nostalgic",
	"Peaceful",
	"Powerful",
	"Refreshing",
	"Romantic",
	"Seductive",
	"Sensual",
	"Serendipitous",
	"Soothing",
	"Sprightly",
	"Sultry",
	"Thrilling",
	"Tranquil",
	"Unpredictable",
	"Vibrant",
	"Whimsical",
	"Zany",
	"Adventurous",
	"Blissful",
	"Chirpy",
	"Ecstatic",
	"Exhilarating",
	"Festive",
	"Flamboyant",
	"Glamorous",
	"Glistening",
	"Insane",
	"Jazzy",
	"Juicy",
	"Magical",
	"Majestic",
	"Miraculous",
	"Mysterious",
	"Precious",
	"Radiant",
	"Sassy",
	"Sunny",
}
