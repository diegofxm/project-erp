package application

import (
	"errors"

	"github.com/diegofxm/erp/internal/thirdparty/domain"
)

// notFoundForRole traduce el "no encontrado" genérico del repositorio al error específico del
// catálogo (cliente/proveedor) que está consultando el caso de uso — mantiene el mismo
// comportamiento de error que tenían los antiguos ErrCustomerNotFound/ErrSupplierNotFound.
func notFoundForRole(err error, role domain.Role) error {
	if !errors.Is(err, domain.ErrPartyNotFound) {
		return err
	}
	if role == domain.RoleSupplier {
		return domain.ErrSupplierNotFound
	}
	return domain.ErrCustomerNotFound
}
