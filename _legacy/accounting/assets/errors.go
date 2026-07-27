package assets

import "errors"

var (
	ErrAssetNotFound       = errors.New("activo fijo no encontrado")
	ErrAssetAlreadyDisposed = errors.New("el activo ya fue dado de baja")
	ErrAlreadyDepreciated  = errors.New("el periodo ya tiene una corrida de depreciación completada")
	ErrNoAssetsToDepreciate = errors.New("no hay activos activos con depreciación pendiente en este periodo")
	ErrProceedsAccountRequired = errors.New("se requiere cuenta de ingresos cuando los proceeds > 0")
	ErrInvalidAcquisitionCost  = errors.New("el costo de adquisición debe ser mayor a cero")
	ErrInvalidSalvageValue     = errors.New("el valor residual no puede ser mayor o igual al costo de adquisición")
	ErrMissingAccounts         = errors.New("asset_account, depreciation_account y accumulated_account son obligatorios")
)
