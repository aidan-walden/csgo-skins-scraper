package main

import "encoding/json"

type Skin struct {
	Name     string              `json:"name"`
	Img      *string             `json:"img"`
	Rarity   SkinRarity          `json:"rarity"`
	Stattrak bool                `json:"stattrak"`
	Pricing  map[string]*float64 `json:"pricing"`
	MinWear  *float64            `json:"minWear"`
	MaxWear  *float64            `json:"maxWear"`
}

type SkinRarity int

const (
	SkinRarityUnknown SkinRarity = iota
	SkinRarityBlue
	SkinRarityPurple
	SkinRarityPink
	SkinRarityRed
	SkinRarityGold
)

func (r SkinRarity) String() string {
	switch r {
	case SkinRarityBlue:
		return "Blue"
	case SkinRarityPurple:
		return "Purple"
	case SkinRarityPink:
		return "Pink"
	case SkinRarityRed:
		return "Red"
	case SkinRarityGold:
		return "Gold"
	default:
		return "Unknown"
	}
}

func (r SkinRarity) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

type CaseResult struct {
	Pink    []*Skin `json:"pink"`
	Red     []*Skin `json:"red"`
	Purple  []*Skin `json:"purple"`
	Blue    []*Skin `json:"blue"`
	Gold    []*Skin `json:"gold"`
	Unknown []*Skin `json:"unknown"`
	Price   float64 `json:"price"`
}

func NewCaseResult() *CaseResult {
	return &CaseResult{
		Pink:    make([]*Skin, 0),
		Red:     make([]*Skin, 0),
		Purple:  make([]*Skin, 0),
		Blue:    make([]*Skin, 0),
		Gold:    make([]*Skin, 0),
		Unknown: make([]*Skin, 0),
	}
}

func (c *CaseResult) AddSkin(skin *Skin) {
	switch skin.Rarity {
	case SkinRarityPink:
		c.Pink = append(c.Pink, skin)
	case SkinRarityRed:
		c.Red = append(c.Red, skin)
	case SkinRarityPurple:
		c.Purple = append(c.Purple, skin)
	case SkinRarityBlue:
		c.Blue = append(c.Blue, skin)
	case SkinRarityGold:
		c.Gold = append(c.Gold, skin)
	default:
		c.Unknown = append(c.Unknown, skin)
	}
}
