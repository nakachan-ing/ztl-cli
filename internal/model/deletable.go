package model

type Deletable interface {
	SetDeleted()
	ResetDeleted()
}
