package tui

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestRowMutationsBlockedInReadOnlyMode(t *testing.T) {
	rows := &fakeRowAdmin{}
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.row = rows
	})

	m, _ = press(t, m, "+")
	if m.modal != modalNone {
		t.Fatalf("read-only + opened modal %v", m.modal)
	}
	if !strings.Contains(m.message, "read-write mode") {
		t.Fatalf("insert message = %q", m.message)
	}
	m, _ = press(t, m, "-")
	if m.modal != modalNone {
		t.Fatalf("read-only - opened modal %v", m.modal)
	}
	if !strings.Contains(m.message, "read-write mode") {
		t.Fatalf("delete message = %q", m.message)
	}
	if rows.calls != 0 {
		t.Fatalf("read-only row change reached the engine: %d calls", rows.calls)
	}
}

func TestRowInsertSQLPassesValuesAndRefreshesData(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	rows := &fakeRowAdmin{result: domain.MutationResult{
		Statement: `INSERT INTO "public"."customers" ("id", "name") VALUES ($1, $2)`,
		Affected:  1,
	}}
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.data = reader
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)
	before := len(reader.calls)

	m, _ = press(t, m, "+")
	if m.modal != modalRowInsert || m.rowInsert.mongo {
		t.Fatalf("modal=%v mongo=%v", m.modal, m.rowInsert.mongo)
	}
	if len(m.rowInsert.columns) != 3 {
		t.Fatalf("columns = %+v", m.rowInsert.columns)
	}
	if m.rowInsert.modes[0] != rowDefault {
		t.Fatalf("initial mode = %v, want default", m.rowInsert.modes[0])
	}
	m = typeString(t, m, "7")
	if m.rowInsert.modes[0] != rowValue {
		t.Fatalf("typing did not switch to value: %v", m.rowInsert.modes[0])
	}
	m, _ = press(t, m, "tab")
	m = typeString(t, m, "Ada")
	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "ctrl+d")
	if m.rowInsert.modes[2] != rowDefault {
		t.Fatalf("mode = %v, want default", m.rowInsert.modes[2])
	}

	m, cmd := press(t, m, "enter")
	if m.modal != modalNone {
		t.Fatalf("submit left modal %v", m.modal)
	}
	m = runCmd(t, m, cmd)
	if rows.calls != 1 || len(rows.inserts) != 1 {
		t.Fatalf("row admin calls=%d inserts=%d", rows.calls, len(rows.inserts))
	}
	spec := rows.inserts[0]
	if spec.Ref.Kind != domain.KindTable || spec.Ref.Name == "" {
		t.Fatalf("insert ref = %+v", spec.Ref)
	}
	if len(spec.Values) != 3 {
		t.Fatalf("values = %+v", spec.Values)
	}
	if spec.Values[0].Column != "id" || spec.Values[0].Value != "7" || spec.Values[0].Null || spec.Values[0].Default {
		t.Fatalf("id value = %+v", spec.Values[0])
	}
	if spec.Values[1].Column != "name" || spec.Values[1].Value != "Ada" {
		t.Fatalf("name value = %+v", spec.Values[1])
	}
	if !spec.Values[2].Default || spec.Values[2].Null {
		t.Fatalf("status value = %+v", spec.Values[2])
	}
	if !strings.Contains(m.message, "row inserted") {
		t.Fatalf("message = %q", m.message)
	}
	if len(reader.calls) != before+1 {
		t.Fatalf("data refresh calls = %d, want %d", len(reader.calls), before+1)
	}
	if !m.data.loaded || m.data.loading {
		t.Fatalf("data not refreshed: loaded=%v loading=%v", m.data.loaded, m.data.loading)
	}
}

func TestRowInsertMongoDocument(t *testing.T) {
	reader := mongoRowReader()
	rows := &fakeRowAdmin{result: domain.MutationResult{
		Statement: `db.getSiblingDB("analytics").getCollection("customers").insertOne(<document>)`,
		Affected:  1,
	}}
	m, _ := dataWorkspaceWith(t, "analytics", func(e *testEnv) {
		e.connector.data = reader
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)

	m, _ = press(t, m, "+")
	if m.modal != modalRowInsert || !m.rowInsert.mongo {
		t.Fatalf("modal=%v mongo=%v", m.modal, m.rowInsert.mongo)
	}
	m.rowInsert.document = `{"name":"Ada","score":1}`
	m.rowInsert.docPos = len([]rune(m.rowInsert.document))
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if rows.calls != 1 || len(rows.inserts) != 1 {
		t.Fatalf("row admin calls=%d", rows.calls)
	}
	spec := rows.inserts[0]
	if spec.Ref.Kind != domain.KindCollection || spec.Ref.Database != "analytics" {
		t.Fatalf("ref = %+v", spec.Ref)
	}
	if spec.Document != `{"name":"Ada","score":1}` {
		t.Fatalf("document = %q", spec.Document)
	}
	if !strings.Contains(m.message, "row inserted") {
		t.Fatalf("message = %q", m.message)
	}
}

func TestRowDeleteRequiresConfirmationAndRefreshesData(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	rows := &fakeRowAdmin{result: domain.MutationResult{
		Statement: `DELETE FROM "public"."customers" WHERE "id" = $1`,
		Affected:  1,
	}}
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.data = reader
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)
	before := len(reader.calls)

	m, _ = press(t, m, "-")
	if m.modal != modalRowDelete {
		t.Fatalf("modal = %v", m.modal)
	}
	if rows.calls != 0 {
		t.Fatal("delete ran before confirmation")
	}
	view := StripANSI(m.viewModal())
	for _, want := range []string{"Delete row from public.customers?", "id=1", "Enter remove"} {
		if !strings.Contains(view, want) {
			t.Fatalf("confirmation missing %q:\n%s", want, view)
		}
	}
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if rows.calls != 1 || len(rows.deletes) != 1 {
		t.Fatalf("row admin calls=%d deletes=%d", rows.calls, len(rows.deletes))
	}
	spec := rows.deletes[0]
	if len(spec.Rows) != 1 || len(spec.Rows[0]) != 1 || spec.Rows[0][0].Column != "id" || spec.Rows[0][0].Value != "1" {
		t.Fatalf("delete keys = %+v", spec.Rows)
	}
	if !strings.Contains(m.message, "row removed") {
		t.Fatalf("message = %q", m.message)
	}
	if len(reader.calls) != before+1 {
		t.Fatalf("data refresh calls = %d, want %d", len(reader.calls), before+1)
	}
}

func TestRowDeleteMongoUsesID(t *testing.T) {
	rows := &fakeRowAdmin{result: domain.MutationResult{
		Statement: `db.getSiblingDB("analytics").getCollection("customers").deleteOne({_id:<key>})`,
		Affected:  1,
	}}
	m, _ := dataWorkspaceWith(t, "analytics", func(e *testEnv) {
		e.connector.data = mongoRowReader()
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)

	m, _ = press(t, m, "-")
	if m.modal != modalRowDelete {
		t.Fatalf("modal = %v", m.modal)
	}
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if rows.calls != 1 || len(rows.deletes) != 1 {
		t.Fatalf("row admin calls=%d", rows.calls)
	}
	keys := rows.deletes[0].Rows
	if len(keys) != 1 || len(keys[0]) != 1 || keys[0][0].Column != "_id" || keys[0][0].Value != "68f000000000000000000001" {
		t.Fatalf("delete keys = %+v", keys)
	}
}

func TestRowDeleteBlockedWithoutStableKey(t *testing.T) {
	rows := &fakeRowAdmin{}
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.data = &fakeDataReader{total: 5, noOrder: true}
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)

	m, _ = press(t, m, "-")
	if m.modal != modalNone {
		t.Fatalf("modal = %v", m.modal)
	}
	if !strings.Contains(m.message, "without a primary key or _id") {
		t.Fatalf("message = %q", m.message)
	}
	if rows.calls != 0 {
		t.Fatal("delete reached the engine without a key")
	}
}

func TestRowDeleteBlockedForBinaryKey(t *testing.T) {
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.data = &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
			return domain.DataPage{
				Grid: domain.Grid{
					Columns: []domain.Column{{Name: "id"}, {Name: "name"}},
					Rows: []domain.Row{{Cells: []domain.Cell{
						{Text: "<binary 16 bytes>", Kind: domain.CellBinary},
						{Text: "Ada", Kind: domain.CellText},
					}}},
				},
				OrderKey:      "id",
				OrderKeyKnown: true,
			}, nil
		}}
	})
	m = enableReadWrite(t, m)

	m, _ = press(t, m, "-")
	if m.modal != modalNone {
		t.Fatalf("modal = %v", m.modal)
	}
	if !strings.Contains(m.message, "binary key") {
		t.Fatalf("message = %q", m.message)
	}
}

func TestRowEditCellUpdatesOneRow(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	rows := &fakeRowAdmin{result: domain.MutationResult{
		Statement: `UPDATE "public"."customers" SET "name" = $1 WHERE "id" = $2`,
		Affected:  1,
	}}
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.data = reader
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)
	before := len(reader.calls)

	m, _ = press(t, m, "right")
	m, _ = press(t, m, "e")
	if m.modal != modalRowEdit {
		t.Fatalf("modal = %v", m.modal)
	}
	if m.rowEdit.column != "name" || m.rowEdit.label != "name" {
		t.Fatalf("edit state = %+v", m.rowEdit)
	}
	if len(m.rowEdit.rows) != 1 || m.rowEdit.rows[0][0].Value != "1" {
		t.Fatalf("targets = %+v", m.rowEdit.rows)
	}
	if m.rowEdit.value != "row-001" {
		t.Fatalf("current value = %q", m.rowEdit.value)
	}
	m.rowEdit.value = "updated"
	m.rowEdit.valuePos = len([]rune("updated"))
	m, cmd := press(t, m, "enter")
	if m.modal != modalNone {
		t.Fatalf("submit left modal %v", m.modal)
	}
	m = runCmd(t, m, cmd)
	if rows.calls != 1 || len(rows.updates) != 1 {
		t.Fatalf("calls=%d updates=%d", rows.calls, len(rows.updates))
	}
	spec := rows.updates[0]
	if spec.Column != "name" || spec.Value.Value != "updated" || spec.Value.Null || spec.Value.Default {
		t.Fatalf("update spec = %+v", spec)
	}
	if len(spec.Rows) != 1 || spec.Rows[0][0].Column != "id" || spec.Rows[0][0].Value != "1" {
		t.Fatalf("update keys = %+v", spec.Rows)
	}
	if !strings.Contains(m.message, "row updated") {
		t.Fatalf("message = %q", m.message)
	}
	if len(reader.calls) != before+1 {
		t.Fatalf("refresh calls = %d, want %d", len(reader.calls), before+1)
	}
}

func TestRowEditSupportsNullAndDefault(t *testing.T) {
	rows := &fakeRowAdmin{result: domain.MutationResult{Statement: "UPDATE", Affected: 1}}
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "e")
	m, _ = press(t, m, "ctrl+n")
	if m.rowEdit.mode != rowNull {
		t.Fatalf("mode = %v, want null", m.rowEdit.mode)
	}
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if len(rows.updates) != 1 || !rows.updates[0].Value.Null {
		t.Fatalf("null update = %+v", rows.updates)
	}

	m, _ = press(t, m, "right") // back to name
	m, _ = press(t, m, "e")
	m, _ = press(t, m, "ctrl+d")
	if m.rowEdit.mode != rowDefault {
		t.Fatalf("mode = %v, want default", m.rowEdit.mode)
	}
	m, cmd = press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if len(rows.updates) != 2 || !rows.updates[1].Value.Default {
		t.Fatalf("default update = %+v", rows.updates)
	}
}

func TestRowEditBlocksPrimaryKey(t *testing.T) {
	rows := &fakeRowAdmin{}
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)

	m, _ = press(t, m, "e")
	if m.modal != modalNone {
		t.Fatalf("modal = %v", m.modal)
	}
	if !strings.Contains(m.message, "primary key") {
		t.Fatalf("message = %q", m.message)
	}
	if rows.calls != 0 {
		t.Fatal("key edit reached the engine")
	}
}

func TestRowSelectionBatchDelete(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	rows := &fakeRowAdmin{result: domain.MutationResult{
		Statement: `DELETE FROM "public"."customers" WHERE ("id" = $1) OR ("id" = $2)`,
		Affected:  2,
	}}
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.data = reader
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)

	m, _ = press(t, m, "space")
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "space")
	if m.data.selectedCount() != 2 {
		t.Fatalf("selected = %d", m.data.selectedCount())
	}
	if !strings.Contains(StripANSI(m.View().Content), "2 selected") {
		t.Fatal("selection banner missing")
	}

	before := len(reader.calls)
	m, _ = press(t, m, "-")
	if m.modal != modalRowDelete {
		t.Fatalf("modal = %v", m.modal)
	}
	view := StripANSI(m.viewModal())
	for _, want := range []string{"Delete 2 rows from public.customers?", "id=1", "id=2"} {
		if !strings.Contains(view, want) {
			t.Fatalf("confirmation missing %q:\n%s", want, view)
		}
	}
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if len(rows.deletes) != 1 || len(rows.deletes[0].Rows) != 2 {
		t.Fatalf("delete spec = %+v", rows.deletes)
	}
	if !strings.Contains(m.message, "2 rows removed") {
		t.Fatalf("message = %q", m.message)
	}
	if len(reader.calls) != before+1 {
		t.Fatalf("refresh calls = %d, want %d", len(reader.calls), before+1)
	}
	if m.data.selectedCount() != 0 {
		t.Fatalf("selection survived refresh: %d", m.data.selectedCount())
	}
}

func TestRowSelectionBatchUpdate(t *testing.T) {
	rows := &fakeRowAdmin{result: domain.MutationResult{
		Statement: `UPDATE "public"."customers" SET "name" = $1 WHERE ("id" = $2) OR ("id" = $3)`,
		Affected:  2,
	}}
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)

	m, _ = press(t, m, "space")
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "space")
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "e")
	if m.modal != modalRowEdit || len(m.rowEdit.rows) != 2 {
		t.Fatalf("modal=%v edit=%+v", m.modal, m.rowEdit)
	}
	if !strings.Contains(StripANSI(m.viewModal()), "2 rows") {
		t.Fatal("multi-row header missing")
	}
	m.rowEdit.value = "batch"
	m.rowEdit.valuePos = len([]rune("batch"))
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if len(rows.updates) != 1 {
		t.Fatalf("updates = %+v", rows.updates)
	}
	spec := rows.updates[0]
	if spec.Column != "name" || spec.Value.Value != "batch" || len(spec.Rows) != 2 {
		t.Fatalf("update spec = %+v", spec)
	}
	if !strings.Contains(m.message, "2 rows updated") {
		t.Fatalf("message = %q", m.message)
	}
}

func TestRowSelectionEscClears(t *testing.T) {
	m, _ := dataWorkspaceWith(t, "prod-ro", nil)
	m, _ = press(t, m, "space")
	if m.data.selectedCount() != 1 {
		t.Fatalf("selected = %d", m.data.selectedCount())
	}
	m, _ = press(t, m, "esc")
	if m.data.selectedCount() != 0 || !strings.Contains(m.message, "selection cleared") {
		t.Fatalf("selected=%d message=%q", m.data.selectedCount(), m.message)
	}
}

func TestRowSelectionBlockedWithoutStableKey(t *testing.T) {
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.data = &fakeDataReader{total: 5, noOrder: true}
	})
	m, _ = press(t, m, "space")
	if m.data.selectedCount() != 0 {
		t.Fatalf("selected = %d", m.data.selectedCount())
	}
	if !strings.Contains(m.message, "without a primary key or _id") {
		t.Fatalf("message = %q", m.message)
	}
}

func TestRowEditMongoField(t *testing.T) {
	rows := &fakeRowAdmin{result: domain.MutationResult{
		Statement: `db.getSiblingDB("analytics").getCollection("customers").updateOne({_id:<key>}, {$set:{"name":<value>}})`,
		Affected:  1,
	}}
	m, _ := dataWorkspaceWith(t, "analytics", func(e *testEnv) {
		e.connector.data = mongoRowReader()
		e.connector.row = rows
	})
	m = enableReadWrite(t, m)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "e")
	if m.modal != modalRowEdit || !m.rowEdit.mongo || m.rowEdit.column != "name" {
		t.Fatalf("modal=%v edit=%+v", m.modal, m.rowEdit)
	}
	m.rowEdit.value = "Grace"
	m.rowEdit.valuePos = len([]rune("Grace"))
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if len(rows.updates) != 1 || rows.updates[0].Value.Value != "Grace" || rows.updates[0].Column != "name" {
		t.Fatalf("update = %+v", rows.updates)
	}
}

func mongoRowReader() *fakeDataReader {
	return &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "_id"}, {Name: "name"}},
				Rows: []domain.Row{{Cells: []domain.Cell{
					{Text: "68f000000000000000000001", Kind: domain.CellText},
					{Text: "Ada", Kind: domain.CellText},
				}}},
			},
			EstimatedTotal: 1,
			EstimateSource: domain.EstimateCollection,
			OrderKey:       "_id",
			OrderKeyKnown:  true,
		}, nil
	}}
}
