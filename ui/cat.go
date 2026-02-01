package ui

import (
	"math/rand"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Cat ASCII art variations
var catArts = []string{
	`
    /\_/\  
   ( o.o ) 
    > ^ <
   /|   |\
  (_|   |_)`,

	`
   /\_____/\
  /  o   o  \
 ( ==  ^  == )
  )         (
 (           )
( (  )   (  ) )
(__(__)___(__)__)`,

	`
      /\___/\
     (  o o  )
     (  =^=  ) 
      (---)
     /|   |\
    (_|   |_)`,

	`
  ╱|、
(˚ˎ 。7  
 |、˜〵          
 じしˍ,)ノ`,

	`
   ∧,,,∧
 ( ̳• · • ̳)
 /    づ♡`,

	`
    /\     /\
   {  '---'  }
   {  O   O  }
   ~~>  V  <~~
      \  \|
       '---'\
       /     \   
      /       '--'
     {        }
      \      /
       '.__.'`,
}

// Encouraging messages the cat can say
var catMessages = []string{
	"Great job! Your code is saved! ✨",
	"Purrfect commit! You're doing amazing! 🌟",
	"Meow! Another save in the bag! 🎉",
	"You're on a roll! Keep vibing! 💫",
	"Nice work, hooman! *purrs* 😸",
	"Commit complete! You're crushing it! 🚀",
	"Saved! Time for a treat break? 🍪",
	"Your code is safe with me! *nuzzles* 💕",
	"Another one! You're unstoppable! ⚡",
	"Meowvelous work! Keep it up! 🌈",
	"*happy cat noises* Great save! 😻",
	"You did it! I believe in you! 💪",
	"Pawsitively amazing commit! 🐾",
	"Your code sparks joy! ✨",
	"Fantastic! You're a coding wizard! 🧙",
}

// GetRandomCat returns a random cat ASCII art
func GetRandomCat() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return catArts[rng.Intn(len(catArts))]
}

// GetRandomCatMessage returns a random encouraging message
func GetRandomCatMessage() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return catMessages[rng.Intn(len(catMessages))]
}

// RenderCelebrationCat renders a cute cat with an encouraging message
func RenderCelebrationCat() string {
	cat := GetRandomCat()
	message := GetRandomCatMessage()

	// Style the cat with the accent color
	catStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	// Style the speech bubble
	bubbleStyle := lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Italic(true).
		PaddingLeft(2)

	// Build the display
	styledCat := catStyle.Render(cat)
	styledMessage := bubbleStyle.Render("💬 " + message)

	return styledCat + "\n\n" + styledMessage
}

