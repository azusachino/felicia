package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

const siteSettingsColumns = `journal_id, title, description, design, default_language, default_theme, accent, created_at, updated_at`

// GetSiteSettings retrieves the identity/style settings saved for one
// journal. Returns domain.ErrNotFound when no row has been saved yet —
// callers fall back to domain.DefaultSiteSettings.
func (r *Repository) GetSiteSettings(ctx context.Context, journalID uuid.UUID) (*domain.SiteSettings, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+siteSettingsColumns+" FROM tb_site_settings WHERE journal_id = ?", idString(journalID))
	settings, err := scanSiteSettings(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get site settings %s: %w", journalID, err)
	}
	return settings, nil
}

// UpsertSiteSettings replaces the single settings row for a journal.
func (r *Repository) UpsertSiteSettings(ctx context.Context, settings *domain.SiteSettings) error {
	if settings == nil || settings.JournalID == uuid.Nil {
		return errors.New("site settings journal ID is required")
	}
	ts := now()
	_, err := r.db.ExecContext(ctx, `INSERT INTO tb_site_settings(journal_id, title, description, design, default_language, default_theme, accent, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(journal_id) DO UPDATE SET
  title=excluded.title,
  description=excluded.description,
  design=excluded.design,
  default_language=excluded.default_language,
  default_theme=excluded.default_theme,
  accent=excluded.accent,
  updated_at=excluded.updated_at`,
		idString(settings.JournalID), settings.Title, settings.Description, settings.Design,
		settings.DefaultLanguage, settings.DefaultTheme, settings.Accent, ts, ts)
	if err != nil {
		return fmt.Errorf("upsert site settings %s: %w", settings.JournalID, err)
	}
	return nil
}

// scanSiteSettings reads one row via the shared scanner interface (store.go),
// used identically by *sql.Row and *sql.Rows.
func scanSiteSettings(row scanner) (*domain.SiteSettings, error) {
	var rawJournalID, title, description, design, defaultLanguage, defaultTheme, accent, created, updated string
	if err := row.Scan(&rawJournalID, &title, &description, &design, &defaultLanguage, &defaultTheme, &accent, &created, &updated); err != nil {
		return nil, err
	}
	journalID, err := parseID(rawJournalID)
	if err != nil {
		return nil, err
	}
	return &domain.SiteSettings{
		JournalID:       journalID,
		Title:           title,
		Description:     description,
		Design:          design,
		DefaultLanguage: defaultLanguage,
		DefaultTheme:    defaultTheme,
		Accent:          accent,
		CreatedAt:       readTime(sql.NullString{String: created, Valid: true}),
		UpdatedAt:       readTime(sql.NullString{String: updated, Valid: true}),
	}, nil
}
