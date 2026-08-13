package dto

import (
	"strings"
	"testing"

	"rms-backend/internal/utils"
)

func flexInt(v int) utils.FlexInt { return utils.FlexInt(v) }

func validPOItem() CreatePOItemRequest {
	return CreatePOItemRequest{
		Product: "Kertas A4",
		Qty:     flexInt(10),
		UnitID:  flexInt(1),
		Price:   utils.FlexFloat(50000),
	}
}

func TestCreatePOItemRequest_Valid(t *testing.T) {
	if err := validPOItem().Validate(); err != nil {
		t.Errorf("expected valid item to pass, got error: %v", err)
	}
}

func TestCreatePOItemRequest_MissingProduct(t *testing.T) {
	item := validPOItem()
	item.Product = ""
	if err := item.Validate(); err == nil {
		t.Error("expected error for empty product name")
	}
}

func TestCreatePOItemRequest_ZeroQty(t *testing.T) {
	item := validPOItem()
	item.Qty = flexInt(0)
	if err := item.Validate(); err == nil {
		t.Error("expected error for qty=0")
	}
}

func TestCreatePOItemRequest_NegativeQty(t *testing.T) {
	item := validPOItem()
	item.Qty = flexInt(-5)
	if err := item.Validate(); err == nil {
		t.Error("expected error for negative qty")
	}
}

func TestCreatePOItemRequest_ZeroUnitID(t *testing.T) {
	item := validPOItem()
	item.UnitID = flexInt(0)
	if err := item.Validate(); err == nil {
		t.Error("expected error for unit_id=0")
	}
}

func TestCreatePOItemRequest_NegativePrice(t *testing.T) {
	item := validPOItem()
	item.Price = utils.FlexFloat(-1000)
	if err := item.Validate(); err == nil {
		t.Error("expected error for negative price")
	}
}

func TestCreatePOItemRequest_ZeroPriceAllowed(t *testing.T) {
	// Harga 0 valid (mis. item gratis/bonus) - hanya harga NEGATIF yang ditolak.
	item := validPOItem()
	item.Price = utils.FlexFloat(0)
	if err := item.Validate(); err != nil {
		t.Errorf("expected zero price to be allowed, got error: %v", err)
	}
}

func validCreatePORequest() CreatePORequest {
	return CreatePORequest{
		OrderNum: "PO-2026-0001",
		RegionID: flexInt(1),
		PicID:    "ADM001",
		PpnID:    flexInt(1),
		Date:     "2026-07-16",
		ClientID: flexInt(1),
		Items:    []CreatePOItemRequest{validPOItem()},
	}
}

func TestCreatePORequest_Valid(t *testing.T) {
	if err := validCreatePORequest().Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestCreatePORequest_MissingOrderNum(t *testing.T) {
	req := validCreatePORequest()
	req.OrderNum = ""
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing order_num")
	}
}

func TestCreatePORequest_MissingRegion(t *testing.T) {
	req := validCreatePORequest()
	req.RegionID = flexInt(0)
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing region_id")
	}
}

func TestCreatePORequest_MissingPIC(t *testing.T) {
	req := validCreatePORequest()
	req.PicID = ""
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing pic_id")
	}
}

func TestCreatePORequest_NoItems(t *testing.T) {
	req := validCreatePORequest()
	req.Items = nil
	if err := req.Validate(); err == nil {
		t.Error("expected error when items is empty")
	}
}

func TestCreatePORequest_InvalidNestedItem(t *testing.T) {
	// Error dari item ke-2 harus ikut ter-propagate dan diberi label "item ke-2".
	req := validCreatePORequest()
	badItem := validPOItem()
	badItem.Product = ""
	req.Items = []CreatePOItemRequest{validPOItem(), badItem}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected error from invalid second item")
	}
	if !strings.Contains(err.Error(), "item ke-2") {
		t.Errorf("expected error to reference 'item ke-2', got: %v", err)
	}
}

func TestCreatePORequest_DivisionIDPtr_Nil(t *testing.T) {
	req := validCreatePORequest()
	req.DivisionID = nil
	if req.DivisionIDPtr() != nil {
		t.Error("expected nil DivisionIDPtr when DivisionID is nil")
	}
}

func TestCreatePORequest_DivisionIDPtr_Set(t *testing.T) {
	req := validCreatePORequest()
	div := flexInt(3)
	req.DivisionID = &div
	ptr := req.DivisionIDPtr()
	if ptr == nil {
		t.Fatal("expected non-nil DivisionIDPtr")
	}
	if *ptr != 3 {
		t.Errorf("expected 3, got %d", *ptr)
	}
}

func TestCreatePORequest_PicClientIDOptional(t *testing.T) {
	// pic_client_id sengaja TIDAK divalidasi wajib - PO tanpa PIC client
	// harus tetap lolos validasi.
	req := validCreatePORequest()
	req.PicClientID = ""
	if err := req.Validate(); err != nil {
		t.Errorf("expected pic_client_id to be optional, got error: %v", err)
	}
}

func TestUpdatePORequest_Valid(t *testing.T) {
	req := UpdatePORequest{OrderNum: "PO-2026-0001"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestUpdatePORequest_MissingOrderNum(t *testing.T) {
	req := UpdatePORequest{}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing order_num")
	}
}

func TestUpdatePORequest_DivisionIDPtr(t *testing.T) {
	req := UpdatePORequest{OrderNum: "PO-1"}
	if req.DivisionIDPtr() != nil {
		t.Error("expected nil when DivisionID not set")
	}
	div := flexInt(5)
	req.DivisionID = &div
	if ptr := req.DivisionIDPtr(); ptr == nil || *ptr != 5 {
		t.Errorf("expected DivisionIDPtr to return 5, got %v", ptr)
	}
}

func TestPOItemRequest_Valid(t *testing.T) {
	req := POItemRequest{Product: "Tinta", Qty: flexInt(2), UnitID: flexInt(1), Price: utils.FlexFloat(150000)}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid item to pass, got error: %v", err)
	}
}

func TestChangeStatusRequest_ValidStatuses(t *testing.T) {
	for _, s := range []string{"open", "progress", "prepared", "complete", "cancel"} {
		req := ChangeStatusRequest{Status: s}
		if err := req.Validate(); err != nil {
			t.Errorf("expected status %q to be valid, got error: %v", s, err)
		}
	}
}

func TestChangeStatusRequest_InvalidStatus(t *testing.T) {
	req := ChangeStatusRequest{Status: "shipped"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for unrecognized status")
	}
}

func TestChangeStatusRequest_EmptyStatus(t *testing.T) {
	req := ChangeStatusRequest{Status: ""}
	if err := req.Validate(); err == nil {
		t.Error("expected error for empty status")
	}
}

func TestChangePaidRequest_ValidValues(t *testing.T) {
	for _, v := range []string{"yes", "no"} {
		req := ChangePaidRequest{Paid: v}
		if err := req.Validate(); err != nil {
			t.Errorf("expected paid=%q to be valid, got error: %v", v, err)
		}
	}
}

func TestChangePaidRequest_InvalidValue(t *testing.T) {
	req := ChangePaidRequest{Paid: "maybe"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for invalid paid value")
	}
}

func TestUpdateInvoiceRequest_Valid(t *testing.T) {
	req := UpdateInvoiceRequest{Invoice: "INV-0001"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid invoice to pass, got error: %v", err)
	}
}

func TestUpdateInvoiceRequest_Empty(t *testing.T) {
	req := UpdateInvoiceRequest{Invoice: ""}
	if err := req.Validate(); err == nil {
		t.Error("expected error for empty invoice number")
	}
}
