package repository

type ErrNotFound struct{ what string }

func (e ErrNotFound) Error() string { return e.what + " not found" }

type ErrConflict struct{ what string }

func (e ErrConflict) Error() string { return e.what }
