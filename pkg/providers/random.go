package providers

import (
	"crypto/rand"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	mrand "math/rand"
	"sort"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"golang.org/x/xerrors"
)

// RandomNumber is a random number provider
type RandomNumber struct {
	config *resources.RandomNumber
	log    clients.Logger
}

// NewRandomNumber creates a random number provider
func NewRandomNumber(c *resources.RandomNumber, l clients.Logger) *RandomNumber {
	return &RandomNumber{c, l}
}

func (n *RandomNumber) Create() error {
	n.log.Info("Creating random number", "ref", n.config.Metadata().ID)

	rn := mrand.Intn(n.config.Maximum-n.config.Minimum) + n.config.Minimum
	n.log.Debug("Generated random number", "ref", n.config.Metadata().ID, "number", rn)

	n.config.Value = rn

	return nil
}

func (n *RandomNumber) Destroy() error {
	return nil
}

func (n *RandomNumber) Lookup() ([]string, error) {
	return nil, nil
}

func (n *RandomNumber) Refresh() error {
	return nil
}

func (c *RandomNumber) Changed() (bool, error) {
	c.log.Info("Checking changes", "ref", c.config.Name)

	return false, nil
}

// RandomID is a provider for generating random IDs
type RandomID struct {
	config *resources.RandomID
	log    clients.Logger
}

// NewRandomID creates a new random ID
func NewRandomID(r *resources.RandomID, l clients.Logger) *RandomID {
	return &RandomID{r, l}
}

func (r *RandomID) Create() error {
	byteLength := r.config.ByteLength
	bytes := make([]byte, byteLength)

	b, err := crand.Reader.Read(bytes)
	if int64(b) != byteLength {
		return xerrors.Errorf("Unable generate random bytes: %w", err)
	}
	if err != nil {
		return xerrors.Errorf("Unable generate random bytes: %w", err)
	}

	hex := hex.EncodeToString(bytes)

	bigInt := big.Int{}
	bigInt.SetBytes(bytes)
	dec := bigInt.String()

	r.config.Hex = hex
	r.config.Dec = dec

	return nil
}

func (n *RandomID) Destroy() error {
	return nil
}

func (n *RandomID) Lookup() ([]string, error) {
	return nil, nil
}

func (n *RandomID) Refresh() error {
	return nil
}

func (c *RandomID) Changed() (bool, error) {
	c.log.Info("Checking changes", "ref", c.config.Name)

	return false, nil
}

// RandomPassword is a provider for generating random passwords
type RandomPassword struct {
	config *resources.RandomPassword
	log    clients.Logger
}

// NewRandomPassword creates a new random password
func NewRandomPassword(r *resources.RandomPassword, l clients.Logger) *RandomPassword {
	return &RandomPassword{r, l}
}

func (r *RandomPassword) Create() error {
	const numChars = "0123456789"
	const lowerChars = "abcdefghijklmnopqrstuvwxyz"
	const upperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var specialChars = "!@#$%&*()-_=+[]{}<>:?"
	var result []byte

	if r.config.OverrideSpecial != "" {
		specialChars = r.config.OverrideSpecial
	}

	var chars = ""
	if *r.config.Upper {
		chars += upperChars
	}

	if *r.config.Lower {
		chars += lowerChars
	}

	if *r.config.Numeric {
		chars += numChars
	}

	if *r.config.Special {
		chars += specialChars
	}

	minMapping := map[string]int64{
		numChars:     r.config.MinNumeric,
		lowerChars:   r.config.MinLower,
		upperChars:   r.config.MinUpper,
		specialChars: r.config.MinSpecial,
	}

	result = make([]byte, 0, r.config.Length)

	for k, v := range minMapping {
		s, err := generateRandomBytes(&k, v)
		if err != nil {
			return err
		}
		result = append(result, s...)
	}

	s, err := generateRandomBytes(&chars, r.config.Length-int64(len(result)))
	if err != nil {
		return err
	}

	result = append(result, s...)

	order := make([]byte, len(result))
	if _, err := rand.Read(order); err != nil {
		return err
	}

	sort.Slice(result, func(i, j int) bool {
		return order[i] < order[j]
	})

	r.config.Value = string(result)

	return nil
}

func (r *RandomPassword) Destroy() error {
	return nil
}

func (r *RandomPassword) Lookup() ([]string, error) {
	return nil, nil
}

func (r *RandomPassword) Refresh() error {
	return nil
}

func (c *RandomPassword) Changed() (bool, error) {
	c.log.Info("Checking changes", "ref", c.config.Name)

	return false, nil
}

// RandomUUID is a provider for generating random UUIDs
type RandomUUID struct {
	config *resources.RandomUUID
	log    clients.Logger
}

// NewRandomUUID creates a new random UUID
func NewRandomUUID(r *resources.RandomUUID, l clients.Logger) *RandomUUID {
	return &RandomUUID{r, l}
}

func (r *RandomUUID) Create() error {
	result, err := uuid.GenerateUUID()
	if err != nil {
		return err
	}

	r.config.Value = result

	return nil
}

func (r *RandomUUID) Destroy() error {
	return nil
}

func (r *RandomUUID) Lookup() ([]string, error) {
	return nil, nil
}

func (r *RandomUUID) Refresh() error {
	return nil
}

func (c *RandomUUID) Changed() (bool, error) {
	c.log.Info("Checking changes", "ref", c.config.Name)

	return false, nil
}

func generateRandomBytes(charSet *string, length int64) ([]byte, error) {
	bytes := make([]byte, length)
	if len(*charSet) == 0 {
		return bytes, nil
	}

	setLen := big.NewInt(int64(len(*charSet)))
	for i := range bytes {
		idx, err := rand.Int(rand.Reader, setLen)
		if err != nil {
			return nil, err
		}
		bytes[i] = (*charSet)[idx.Int64()]
	}
	return bytes, nil
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

// RandomCreature is a provider for generating random creatures
type RandomCreature struct {
	config *resources.RandomCreature
	log    clients.Logger
}

// NewRandomCreature creates a new random Creature
func NewRandomCreature(r *resources.RandomCreature, l clients.Logger) *RandomCreature {
	return &RandomCreature{r, l}
}

func (r *RandomCreature) Create() error {
	ci := mrand.Intn(99)
	ai := mrand.Intn(99)

	r.config.Value = fmt.Sprintf("%s-%s", adjectives[ai], creatures[ci])

	return nil
}

func (r *RandomCreature) Destroy() error {
	return nil
}

func (r *RandomCreature) Lookup() ([]string, error) {
	return nil, nil
}

func (r *RandomCreature) Refresh() error {
	return nil
}

func (c *RandomCreature) Changed() (bool, error) {
	c.log.Info("Checking changes", "ref", c.config.Name)

	return false, nil
}
