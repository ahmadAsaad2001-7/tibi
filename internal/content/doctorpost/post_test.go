package doctorpost

import (
	"testing"
	"time"
)

func TestPublishAndUnpublish(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	p := &Post{Title: "tip"}
	if err := p.Publish(now); err != nil {
		t.Fatal(err)
	}
	if !p.IsPublished || p.PublishedAt == nil || !p.PublishedAt.Equal(now) {
		t.Fatalf("published = %+v", p)
	}
	if err := p.Publish(now); err != ErrAlreadyPublished {
		t.Fatalf("second publish = %v", err)
	}
	if err := p.Unpublish(now); err != nil {
		t.Fatal(err)
	}
	if p.IsPublished || p.PublishedAt != nil {
		t.Fatalf("unpublished = %+v", p)
	}
	if err := p.Unpublish(now); err != ErrNotPublished {
		t.Fatalf("second unpublish = %v", err)
	}
}

func TestUpdateLeavesAbsentFields(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	excerpt := "old"
	p := &Post{Title: "old", Excerpt: &excerpt, Type: TypeHealthTip}
	title := "new"
	p.Update(UpdateInput{Title: &title}, now)
	if p.Title != "new" || p.Excerpt == nil || *p.Excerpt != "old" || p.Type != TypeHealthTip {
		t.Fatalf("post = %+v", p)
	}
	cleared := (*string)(nil)
	p.Update(UpdateInput{Excerpt: &cleared}, now)
	if p.Excerpt != nil {
		t.Fatal("excerpt should be cleared")
	}
}
