package db

import (
	"database/sql"
	"fmt"

	"github.com/ramanasai/pulse/internal/encryption"
)

// EncryptionManager handles database-level encryption
type EncryptionManager struct {
	db        *sql.DB
	encryptor *encryption.Encryptor
	enabled   bool
}

// NewEncryptionManager creates a new encryption manager
func NewEncryptionManager(db *sql.DB, password string) (*EncryptionManager, error) {
	if password == "" {
		return &EncryptionManager{db: db, enabled: false}, nil
	}

	encryptor, err := encryption.NewEncryptor(password)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &EncryptionManager{
		db:        db,
		encryptor: encryptor,
		enabled:   true,
	}, nil
}

// AddEncryptedEntry adds an entry with optional encryption
func (em *EncryptionManager) AddEncryptedEntry(text, project, tags, category string, encrypt bool) (int64, error) {
	var encryptedText sql.NullString
	var encryptedProject sql.NullString
	var encryptedTags sql.NullString

	// Encrypt fields if requested and encryption is enabled
	if encrypt && em.enabled {
		if text != "" {
			encText, err := em.encryptor.Encrypt(text)
			if err != nil {
				return 0, fmt.Errorf("failed to encrypt text: %w", err)
			}
			encryptedText = sql.NullString{String: encText, Valid: true}
		}

		if project != "" {
			encProject, err := em.encryptor.Encrypt(project)
			if err != nil {
				return 0, fmt.Errorf("failed to encrypt project: %w", err)
			}
			encryptedProject = sql.NullString{String: encProject, Valid: true}
		}

		if tags != "" {
			encTags, err := em.encryptor.Encrypt(tags)
			if err != nil {
				return 0, fmt.Errorf("failed to encrypt tags: %w", err)
			}
			encryptedTags = sql.NullString{String: encTags, Valid: true}
		}
	} else {
		// No encryption - still use NullString for consistency
		encryptedText = sql.NullString{String: text, Valid: text != ""}
		encryptedProject = sql.NullString{String: project, Valid: project != ""}
		encryptedTags = sql.NullString{String: tags, Valid: tags != ""}
	}

	// Insert entry
	result, err := em.db.Exec(`
		INSERT INTO entries (category, text, project, tags, encrypted)
		VALUES (?, ?, ?, ?, ?)
	`, category, encryptedText, encryptedProject, encryptedTags, encrypt && em.enabled)

	if err != nil {
		return 0, fmt.Errorf("failed to insert entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// DecryptEntry decrypts an entry if needed
func (em *EncryptionManager) DecryptEntry(entry Entry) (Entry, error) {
	if !entry.Encrypted || !em.enabled {
		return entry, nil
	}

	result := entry

	// Decrypt text
	if entry.Text.Valid && entry.Text.String != "" {
		decryptedText, err := em.encryptor.Decrypt(entry.Text.String)
		if err != nil {
			return entry, fmt.Errorf("failed to decrypt text: %w", err)
		}
		result.Text = sql.NullString{String: decryptedText, Valid: true}
	}

	// Decrypt project
	if entry.Project.Valid && entry.Project.String != "" {
		decryptedProject, err := em.encryptor.Decrypt(entry.Project.String)
		if err != nil {
			return entry, fmt.Errorf("failed to decrypt project: %w", err)
		}
		result.Project = sql.NullString{String: decryptedProject, Valid: true}
	}

	// Decrypt tags
	if entry.Tags.Valid && entry.Tags.String != "" {
		decryptedTags, err := em.encryptor.Decrypt(entry.Tags.String)
		if err != nil {
			return entry, fmt.Errorf("failed to decrypt tags: %w", err)
		}
		result.Tags = sql.NullString{String: decryptedTags, Valid: true}
	}

	return result, nil
}

// UpdateEncryptedEntry updates an entry with encryption
func (em *EncryptionManager) UpdateEncryptedEntry(id int, text, project, tags, category string, encrypt bool) error {
	var encryptedText sql.NullString
	var encryptedProject sql.NullString
	var encryptedTags sql.NullString

	// Encrypt fields if requested and encryption is enabled
	if encrypt && em.enabled {
		if text != "" {
			encText, err := em.encryptor.Encrypt(text)
			if err != nil {
				return fmt.Errorf("failed to encrypt text: %w", err)
			}
			encryptedText = sql.NullString{String: encText, Valid: true}
		}

		if project != "" {
			encProject, err := em.encryptor.Encrypt(project)
			if err != nil {
				return fmt.Errorf("failed to encrypt project: %w", err)
			}
			encryptedProject = sql.NullString{String: encProject, Valid: true}
		}

		if tags != "" {
			encTags, err := em.encryptor.Encrypt(tags)
			if err != nil {
				return fmt.Errorf("failed to encrypt tags: %w", err)
			}
			encryptedTags = sql.NullString{String: encTags, Valid: true}
		}
	} else {
		// No encryption - still use NullString for consistency
		encryptedText = sql.NullString{String: text, Valid: text != ""}
		encryptedProject = sql.NullString{String: project, Valid: project != ""}
		encryptedTags = sql.NullString{String: tags, Valid: tags != ""}
	}

	// Update entry
	_, err := em.db.Exec(`
		UPDATE entries
		SET category = ?, text = ?, project = ?, tags = ?, encrypted = ?
		WHERE id = ?
	`, category, encryptedText, encryptedProject, encryptedTags, encrypt && em.enabled, id)

	if err != nil {
		return fmt.Errorf("failed to update entry: %w", err)
	}

	return nil
}

// IsEnabled returns whether encryption is enabled
func (em *EncryptionManager) IsEnabled() bool {
	return em.enabled
}

// EnsureEncryptedColumn ensures the encrypted column exists
func EnsureEncryptedColumn(db *sql.DB) error {
	// Check if encrypted column exists
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pragma_table_info('entries')
			WHERE name = 'encrypted'
		)
	`).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check encrypted column: %w", err)
	}

	if !exists {
		// Add encrypted column
		_, err := db.Exec(`
			ALTER TABLE entries ADD COLUMN encrypted BOOLEAN DEFAULT FALSE
		`)
		if err != nil {
			return fmt.Errorf("failed to add encrypted column: %w", err)
		}
	}

	return nil
}