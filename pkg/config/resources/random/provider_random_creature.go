package random

import (
	"fmt"
	mrand "math/rand"

	htypes "github.com/jumppad-labs/hclconfig/types"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

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

// RandomCreature is a provider for generating random creatures
type RandomCreatureProvider struct {
	config *RandomCreature
	log    sdk.Logger
}

func (p *RandomCreatureProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*RandomCreature)
	if !ok {
		return fmt.Errorf("unable to initialize RandomCreature provider, resource is not of type RandomCreature")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *RandomCreatureProvider) Create() error {
	ci := mrand.Intn(99)
	ai := mrand.Intn(99)

	p.config.Value = fmt.Sprintf("%s-%s", adjectives[ai], creatures[ci])

	return nil
}

func (p *RandomCreatureProvider) Destroy() error {
	return nil
}

func (p *RandomCreatureProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *RandomCreatureProvider) Refresh() error {
	return nil
}

func (p *RandomCreatureProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.ID)

	return false, nil
}
