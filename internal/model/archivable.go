package model

type Archivable interface {
	SetArchived()
	ResetArchived()
}
