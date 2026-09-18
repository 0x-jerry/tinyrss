package repository

import "database/sql"

func (r *Repo) CreateFolder(name string) (Folder, error) {
	var f Folder
	err := r.DB.QueryRow(`INSERT INTO folders(name, sort_order) VALUES (?, COALESCE((SELECT MAX(sort_order)+1 FROM folders), 0)) RETURNING id, name, sort_order`, name).
		Scan(&f.ID, &f.Name, &f.SortOrder)
	if isUnique(err) {
		return f, ErrConflict{what: "folder name already exists"}
	}
	return f, err
}

func (r *Repo) ListFolders() ([]Folder, error) {
	rows, err := r.DB.Query(`SELECT id, name, sort_order FROM folders ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Folder{}
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) GetFolder(id int) (Folder, error) {
	var f Folder
	err := r.DB.QueryRow(`SELECT id, name, sort_order FROM folders WHERE id = ?`, id).
		Scan(&f.ID, &f.Name, &f.SortOrder)
	if err == sql.ErrNoRows {
		return f, ErrNotFound{what: "folder"}
	}
	return f, err
}

func (r *Repo) UpdateFolder(id int, name string) error {
	res, err := r.DB.Exec(`UPDATE folders SET name = ? WHERE id = ?`, name, id)
	if isUnique(err) {
		return ErrConflict{what: "folder name already exists"}
	}
	if err != nil {
		return err
	}
	return requireAffected(res, "folder")
}

func (r *Repo) DeleteFolder(id int) error {
	// ON DELETE SET NULL folds its feeds into uncategorized automatically.
	res, err := r.DB.Exec(`DELETE FROM folders WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res, "folder")
}
