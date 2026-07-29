package hygiene

import (
	"testing"

	"github.com/gofrs/uuid"
)

func ptr(v int64) *int64 { return &v }

func TestDetect_FlagsTestTitlesAndOrganizers(t *testing.T) {
	issues := Detect([]Candidate{
		{EventID: uuid.Must(uuid.NewV4()), Title: "QA Тур в Геленджик (Блок 8)", OrganizerName: "QA Block8"},
		{EventID: uuid.Must(uuid.NewV4()), Title: "bla bla meet", OrganizerName: "kornkorn10"},
		{EventID: uuid.Must(uuid.NewV4()), Title: "Летний фестиваль медиаискусства", OrganizerName: "Музей «Гараж»"},
	})
	if len(issues) != 2 {
		t.Fatalf("issues = %d, want 2", len(issues))
	}
	if issues[0].Kind != KindTestData || issues[0].OrganizerName != "QA Block8" {
		t.Fatalf("issue[0] = %+v", issues[0])
	}
}

func TestDetect_FlagsOrganizerNameEvenWhenTitleIsClean(t *testing.T) {
	issues := Detect([]Candidate{
		{Title: "Лекция о городе", OrganizerName: "Тестовый организатор"},
	})
	if len(issues) != 1 || issues[0].Kind != KindTestData {
		t.Fatalf("issues = %+v, want one test_data", issues)
	}
}

func TestDetect_FlagsSuspiciousPrice(t *testing.T) {
	issues := Detect([]Candidate{
		{Title: "Лабораторная сцена", OrganizerName: "Электротеатр", PriceType: "paid", PriceMin: ptr(100500)},
		{Title: "Концерт", OrganizerName: "Винзавод", PriceType: "paid", PriceMin: ptr(2000)},
	})
	if len(issues) != 1 {
		t.Fatalf("issues = %d, want 1", len(issues))
	}
	if issues[0].Kind != KindSuspiciousPrice || issues[0].PriceRUB == nil || *issues[0].PriceRUB != 100500 {
		t.Fatalf("issue = %+v", issues[0])
	}
}

func TestDetect_IgnoresFreeEventsAndNilPrices(t *testing.T) {
	if got := Detect([]Candidate{
		{Title: "Медиация", OrganizerName: "ГМИИ", PriceType: "free", PriceMin: ptr(999999)},
		{Title: "Читательская группа", OrganizerName: "Дом культуры", PriceType: "paid"},
	}); len(got) != 0 {
		t.Fatalf("issues = %+v, want none", got)
	}
}

func TestDetect_TestDataWinsOverPrice(t *testing.T) {
	issues := Detect([]Candidate{
		{Title: "QA прайс", OrganizerName: "QA Block8", PriceType: "paid", PriceMin: ptr(500000)},
	})
	if len(issues) != 1 || issues[0].Kind != KindTestData {
		t.Fatalf("issues = %+v, want a single test_data issue", issues)
	}
}

func TestReasonFor_IsRussianAndPrefixed(t *testing.T) {
	if got := ReasonFor(KindTestData); got != "Гигиена контента: тестовые данные" {
		t.Fatalf("ReasonFor(test_data) = %q", got)
	}
	if got := ReasonFor(KindSuspiciousPrice); got != "Гигиена контента: подозрительная цена" {
		t.Fatalf("ReasonFor(suspicious_price) = %q", got)
	}
}
