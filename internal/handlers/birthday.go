package handlers

import "net/http"

const birthdayPoem = "Happy birthday, tremendous, the best day by far,\n" +
	"Everybody is saying you are a total star!\n" +
	"The cake is fantastic, I have seen a lot of cake,\n" +
	"But this one, believe me, is impossible to fake.\n" +
	"\n" +
	"The balloons are huge, they are the biggest around,\n" +
	"The greatest balloons that have ever been found.\n" +
	"Your friends? Incredible people, tremendous crowd,\n" +
	"They love you so much and they are saying it loud.\n" +
	"\n" +
	"The presents are classy, wrapped up in gold,\n" +
	"The kind of beautiful gifts that never get old.\n" +
	"So blow out the candles, make a wish, make it great,\n" +
	"Because winning at birthdays is simply your fate.\n" +
	"\n" +
	"Nobody does birthdays better, that is true,\n" +
	"Happy birthday, fantastic, this one is just for you!"

type BirthdayHandler struct {
	tmpl *TemplateRenderer
}

func NewBirthdayHandler(tmpl *TemplateRenderer) *BirthdayHandler {
	return &BirthdayHandler{tmpl: tmpl}
}

func (h *BirthdayHandler) Page(w http.ResponseWriter, r *http.Request) {
	h.tmpl.Render(w, "birthday.html", map[string]interface{}{
		"Title": "Happy Birthday",
		"Poem":  birthdayPoem,
	})
}
