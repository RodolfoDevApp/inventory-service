package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/domain"
)

type PgStockItemRepository struct {
	db *sql.DB
}

func NewPgStockItemRepository(db *sql.DB) *PgStockItemRepository {
	return &PgStockItemRepository{db: db}
}

func (r *PgStockItemRepository) GetBySkus(
	ctx context.Context,
	skus []string,
) (map[string]*domain.StockItem, error) {
	if len(skus) == 0 {
		return map[string]*domain.StockItem{}, nil
	}

	query := `
        select id, sku, available_quantity, reserved_quantity, updated_at_utc
        from inventory_stock_items
        where sku = any($1)
    `
	rows, err := r.db.QueryContext(ctx, query, skus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*domain.StockItem)
	for rows.Next() {
		var item domain.StockItem
		if err := rows.Scan(
			&item.ID,
			&item.Sku,
			&item.Available,
			&item.Reserved,
			&item.UpdatedAtUtc,
		); err != nil {
			return nil, err
		}
		result[item.Sku] = &item
	}

	// missing skus are simply absent
	return result, nil
}

func (r *PgStockItemRepository) UpsertMany(
	ctx context.Context,
	items []*domain.StockItem,
) error {
	if len(items) == 0 {
		return nil
	}

	query := `
        insert into inventory_stock_items (id, sku, available_quantity, reserved_quantity, updated_at_utc)
        values ($1,$2,$3,$4,$5)
        on conflict (sku) do update
        set available_quantity = excluded.available_quantity,
            reserved_quantity = excluded.reserved_quantity,
            updated_at_utc = excluded.updated_at_utc
    `
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}
		if item.UpdatedAtUtc.IsZero() {
			item.UpdatedAtUtc = time.Now().UTC()
		}
		if _, err := stmt.ExecContext(
			ctx,
			item.ID,
			item.Sku,
			item.Available,
			item.Reserved,
			item.UpdatedAtUtc,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Reservations

type PgStockReservationRepository struct {
	db *sql.DB
}

func NewPgStockReservationRepository(db *sql.DB) *PgStockReservationRepository {
	return &PgStockReservationRepository{db: db}
}

func (r *PgStockReservationRepository) GetByOrderID(
	ctx context.Context,
	orderID uuid.UUID,
) (*domain.StockReservation, error) {
	query := `
        select id, order_id, user_id, status, reserved_at_utc, released_at_utc
        from inventory_reservations
        where order_id = $1
    `
	row := r.db.QueryRowContext(ctx, query, orderID)
	var res domain.StockReservation
	var status string
	var releasedAt sql.NullTime
	if err := row.Scan(
		&res.ID,
		&res.OrderID,
		&res.UserID,
		&status,
		&res.ReservedAtUtc,
		&releasedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	res.Status = domain.ReservationStatus(status)
	if releasedAt.Valid {
		t := releasedAt.Time
		res.ReleasedAtUtc = &t
	}

	// Load lines
	lq := `
        select id, sku, quantity
        from inventory_reservation_lines
        where reservation_id = $1
    `
	rows, err := r.db.QueryContext(ctx, lq, res.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := []domain.ReservationLine{}
	for rows.Next() {
		var l domain.ReservationLine
		l.ReservationID = res.ID
		if err := rows.Scan(&l.ID, &l.Sku, &l.Quantity); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	res.Lines = lines
	return &res, nil
}

func (r *PgStockReservationRepository) Insert(
	ctx context.Context,
	res *domain.StockReservation,
) error {
	if res.ID == uuid.Nil {
		res.ID = uuid.New()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := `
        insert into inventory_reservations
        (id, order_id, user_id, status, reserved_at_utc, released_at_utc)
        values ($1,$2,$3,$4,$5,$6)
    `
	var releasedAt *time.Time
	if res.ReleasedAtUtc != nil {
		releasedAt = res.ReleasedAtUtc
	}
	if _, err := tx.ExecContext(
		ctx, q,
		res.ID,
		res.OrderID,
		res.UserID,
		string(res.Status),
		res.ReservedAtUtc,
		releasedAt,
	); err != nil {
		return err
	}

	lq := `
        insert into inventory_reservation_lines
        (id, reservation_id, sku, quantity)
        values ($1,$2,$3,$4)
    `
	for _, l := range res.Lines {
		id := l.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		if _, err := tx.ExecContext(
			ctx, lq,
			id, res.ID, l.Sku, l.Quantity,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PgStockReservationRepository) Update(
	ctx context.Context,
	res *domain.StockReservation,
) error {
	q := `
        update inventory_reservations
        set status = $2,
            released_at_utc = $3
        where id = $1
    `
	_, err := r.db.ExecContext(
		ctx, q,
		res.ID,
		string(res.Status),
		res.ReleasedAtUtc,
	)
	return err
}
