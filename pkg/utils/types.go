package utils

type (
	ScopeAsset struct{
		Type string
		Value string
	}
	
	NewScope struct {
		NewFailTarget  string
		NewAsset    ScopeAsset
		NewPublicURL string
		NewPrivateURL string
	}

	HandleClassifier struct{
		Type string
		Handle string
	}
) 
