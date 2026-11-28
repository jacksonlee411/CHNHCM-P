package postgres

import (
	"context"
	"errors"
	"regexp"
	"strings"

	d "chnhcm.local/chnhcm/internal/contexts/personnel/domain"
	p "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// InitializeAddressTypes runs with the initializer's connection, including on a new instance.
func InitializeAddressTypes(ctx context.Context, tx pgx.Tx, instance string) error {
	// The offline initializer also prepares supported predecessor databases for upgrade.
	var present bool
	if err := tx.QueryRow(ctx, "select to_regprocedure('personnel.initialize_address_types(uuid)') is not null").Scan(&present); err != nil || !present {
		return err
	}
	_, err := tx.Exec(ctx, "select personnel.initialize_address_types($1)", instance)
	return err
}

const typeColumns = "item_id,head,code,name,sort_order,first_day::text,selectable"

func scanAddressType(row pgx.Row) (p.AddressType, error) {
	var x p.AddressType
	err := row.Scan(&x.ID, &x.Revision, &x.Code, &x.Name, &x.Order, &x.FirstDay, &x.Selectable)
	return x, err
}

func (s *Store) AddressTypes(ctx context.Context, c f.Credentials) (p.AddressTypeList, error) {
	return run(ctx, s, c, func(r *request) (p.AddressTypeList, error) {
		var out p.AddressTypeList
		if !r.allowed("address-type.read", "") {
			return out, d.Denied()
		}
		rows, err := r.tx.Query(ctx, "select "+typeColumns+" from personnel.address_type where instance_id=$1 order by sort_order,item_id", r.instance)
		if err != nil {
			return out, err
		}
		out.Items, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (p.AddressType, error) { return scanAddressType(row) })
		out.CanMaintain = r.allowed("address-type.maintain", "")
		out.DefaultFirstDay = f.DefaultEffectiveDate
		return out, err
	})
}
func (s *Store) AddressTypeSource(ctx context.Context, c f.Credentials, id, revision string) (p.AddressTypeSource, error) {
	return run(ctx, s, c, func(r *request) (p.AddressTypeSource, error) {
		var x p.AddressTypeSource
		if !r.allowed("address-type.read", "") {
			return x, d.Denied()
		}
		if !f.ValidID(id) || !f.ValidID(revision) {
			return x, d.Missing()
		}
		err := r.tx.QueryRow(ctx, `select t.item_id,v.revision,t.code,v.name,v.sort_order,t.first_day::text,v.selectable,v.actor_id,v.recorded_at
   from personnel.address_type_revision v join personnel.address_type t using(instance_id,item_id)
   where v.instance_id=$1 and v.item_id=$2 and v.revision=$3`, r.instance, id, revision).Scan(
			&x.ID, &x.Revision, &x.Code, &x.Name, &x.Order, &x.FirstDay, &x.Selectable, &x.ActorID, &x.RecordedAt)
		return x, noRows(err)
	})
}

func (s *Store) AddressTypeCandidates(ctx context.Context, c f.Credentials, person, item string) ([]p.AddressTypeOption, error) {
	return run(ctx, s, c, func(r *request) ([]p.AddressTypeOption, error) {
		ok, err := r.readable(person)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, d.Missing()
		}
		if !r.itemWrite(d.Addresses, person, item, "addressType") {
			return nil, d.Denied()
		}
		if item != "" {
			if !f.ValidID(item) {
				return nil, d.Missing()
			}
			err = r.tx.QueryRow(ctx, `select exists(select from personnel.person p join personnel.profile_item_payload i
    on i.instance_id=p.instance_id and i.person_id=p.person_id and i.version=p.head
    where p.instance_id=$1 and p.person_id=$2 and i.section='addresses' and i.item_id=$3 and i.ordinary_until>$4)`, r.instance, person, item, r.a.At).Scan(&ok)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, d.Missing()
			}
		}
		rows, err := r.tx.Query(ctx, `select item_id,head,code,name from personnel.address_type
    where instance_id=$1 and selectable and first_day<=$2::date order by sort_order,item_id`, r.instance, r.a.At.In(f.Shanghai).Format("2006-01-02"))
		if err != nil {
			return nil, err
		}
		return pgx.CollectRows(rows, func(row pgx.CollectableRow) (p.AddressTypeOption, error) {
			var x p.AddressTypeOption
			err := row.Scan(&x.ID, &x.Revision, &x.Code, &x.Name)
			return x, err
		})
	})
}

func changedType() error {
	return f.Fail(409, "PERSON_ADDRESS_TYPE_CHANGED", "地址类型已变化，请重新选择")
}
func (r *request) adoptAddressType(person string, row *profileItem, choice *p.AddressTypeChoice, legacy string) error {
	var query, key string
	if choice != nil {
		if row.AddressType != nil && row.AddressType.AddressTypeChoice == *choice {
			return nil
		}
		if !f.ValidID(choice.ID) || !f.ValidID(choice.Revision) {
			return d.Invalid("addressType", "请选择有效的地址类型")
		}
		query = "select " + typeColumns + " from personnel.address_type where instance_id=$1 and item_id=$2 for share"
		key = choice.ID
	} else {
		// Only an original system label on its initial revision can stand in for an old client's choice.
		query = "select " + typeColumns + ` from personnel.address_type t where instance_id=$1 and legacy_label=$2 and exists
    (select from personnel.address_type_revision v where (v.instance_id,v.item_id,v.revision)=(t.instance_id,t.item_id,t.head) and actor_id is null) for share of t`
		key = legacy
	}
	x, err := scanAddressType(r.tx.QueryRow(r.ctx, query, r.instance, key))
	if errors.Is(err, pgx.ErrNoRows) {
		if choice == nil {
			return d.Invalid("addressType", "请选择当前可用的地址类型")
		}
		return changedType()
	}
	if err != nil {
		return err
	}
	if err = r.tx.QueryRow(r.ctx, "select clock_timestamp()").Scan(&r.a.At); err != nil {
		return err
	}
	if !r.visible(person) || !r.itemWrite(d.Addresses, person, row.ItemID, "addressType") {
		return d.Denied()
	}
	day := r.a.At.In(f.Shanghai).Format("2006-01-02")
	if !x.Selectable || x.FirstDay > day || choice != nil && x.Revision != choice.Revision {
		return changedType()
	}
	row.Fields["addressType"] = x.Name
	row.AddressType = &p.AddressTypeAdoption{AddressTypeChoice: x.AddressTypeChoice, Date: day}
	return nil
}

var addressTypeCode = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{0,39}$`)

func (r *request) addressTypeReceipt(submission string) (p.AddressType, string, error) {
	var x p.AddressType
	var digest string
	err := r.tx.QueryRow(r.ctx, `select t.item_id,v.revision,t.code,v.name,v.sort_order,t.first_day::text,v.selectable,v.digest
   from personnel.address_type_revision v join personnel.address_type t using(instance_id,item_id)
   where v.instance_id=$1 and v.actor_id=$2 and v.submission_id=$3`, r.instance, r.a.Principal.ID, submission).
		Scan(&x.ID, &x.Revision, &x.Code, &x.Name, &x.Order, &x.FirstDay, &x.Selectable, &digest)
	return x, digest, err
}
func (r *request) lockTypeSubmission(submission string, write bool) error {
	if !f.ValidID(submission) {
		return d.Invalid("submissionId", "提交号无效")
	}
	if err := r.lock("address-type:" + r.a.Principal.ID + ":" + submission); err != nil {
		return err
	}
	if !r.allowed("address-type.read", "") || write && !r.allowed("address-type.maintain", "") {
		return d.Denied()
	}
	return nil
}
func (s *Store) AddressTypeReceipt(ctx context.Context, c f.Credentials, submission string) (p.AddressTypeReceipt, error) {
	return run(ctx, s, c, func(r *request) (p.AddressTypeReceipt, error) {
		var out p.AddressTypeReceipt
		if err := r.lockTypeSubmission(submission, false); err != nil {
			return out, err
		}
		x, _, err := r.addressTypeReceipt(submission)
		if err == nil {
			return p.AddressTypeReceipt{Outcome: "accepted", Item: &x}, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
		_, err = r.tx.Exec(ctx, "insert into personnel.address_type_not_applied values($1,$2,$3) on conflict do nothing", r.instance, r.a.Principal.ID, submission)
		return p.AddressTypeReceipt{Outcome: "not_applied"}, err
	})
}
func (s *Store) SaveAddressType(ctx context.Context, c f.Credentials, in p.AddressTypeCommand) (p.AddressType, error) {
	return run(ctx, s, c, func(r *request) (p.AddressType, error) {
		var zero p.AddressType
		if err := r.lockTypeSubmission(in.SubmissionID, true); err != nil {
			return zero, err
		}
		digest := hash(in)
		x, prior, err := r.addressTypeReceipt(in.SubmissionID)
		if err == nil {
			if digest != prior {
				return zero, d.Conflict()
			}
			return x, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return zero, err
		}
		var closed bool
		if err = r.tx.QueryRow(ctx, "select exists(select from personnel.address_type_not_applied where instance_id=$1 and actor_id=$2 and submission_id=$3)", r.instance, r.a.Principal.ID, in.SubmissionID).Scan(&closed); err != nil {
			return zero, err
		}
		if closed {
			return zero, f.Fail(409, "ADDRESS_TYPE_NOT_APPLIED", "原提交已确认未执行")
		}
		name, err := d.ProfileText("name", in.Name)
		if err != nil {
			return zero, err
		}
		if name == "" {
			return zero, d.Invalid("name", "请填写名称")
		}
		if in.Order < 0 || in.Order > 999999 {
			return zero, d.Invalid("sortOrder", "排序须为 0 至 999999 的整数")
		}
		if in.Selectable == nil {
			return zero, d.Invalid("selectable", "请选择可选状态")
		}
		revision := f.NewID()
		if in.ID == "" {
			code := strings.ToUpper(strings.TrimSpace(in.Code))
			if !addressTypeCode.MatchString(code) {
				return zero, d.Invalid("code", "编码须为 1 至 40 位字母、数字、下划线或连字符，并以字母或数字开头")
			}
			if !f.ValidDate(in.FirstDay) {
				return zero, d.Invalid("firstDay", "请填写有效的初始生效日")
			}
			if in.ExpectedRevision != "" {
				return zero, d.Conflict()
			}
			x, err = scanAddressType(r.tx.QueryRow(ctx, `insert into personnel.address_type(instance_id,code,name,sort_order,first_day,selectable,head)
    values($1,$2,$3,$4,$5,$6,$7) returning `+typeColumns, r.instance, code, name, in.Order, in.FirstDay, *in.Selectable, revision))
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23505" {
				return zero, d.Invalid("code", "此编码已被使用")
			}
		} else {
			if !f.ValidID(in.ID) || !f.ValidID(in.ExpectedRevision) || in.Code != "" || in.FirstDay != "" {
				return zero, d.Invalid("id", "请按原配置项和修订保存")
			}
			x, err = scanAddressType(r.tx.QueryRow(ctx, "select "+typeColumns+" from personnel.address_type where instance_id=$1 and item_id=$2 for update", r.instance, in.ID))
			if err != nil {
				return zero, noRows(err)
			}
			if x.Revision != in.ExpectedRevision {
				return zero, f.Fail(409, "ADDRESS_TYPE_CONFLICT", "此配置项已由其他人修改，请重新核对")
			}
			if x.Name == name && x.Order == in.Order && x.Selectable == *in.Selectable {
				return zero, d.Invalid("name", "没有需要保存的修改")
			}
			x, err = scanAddressType(r.tx.QueryRow(ctx, `update personnel.address_type set head=$3,name=$4,sort_order=$5,selectable=$6
    where instance_id=$1 and item_id=$2 returning `+typeColumns, r.instance, in.ID, revision, name, in.Order, *in.Selectable))
		}
		if err != nil {
			return zero, err
		}
		if err = r.tx.QueryRow(ctx, "select clock_timestamp()").Scan(&r.a.At); err != nil {
			return zero, err
		}
		if !r.allowed("address-type.read", "") || !r.allowed("address-type.maintain", "") {
			return zero, d.Denied()
		}
		_, err = r.tx.Exec(ctx, `insert into personnel.address_type_revision(instance_id,item_id,revision,name,sort_order,selectable,actor_id,submission_id,digest)
    values($1,$2,$3,$4,$5,$6,$7,$8,$9)`, r.instance, x.ID, x.Revision, x.Name, x.Order, x.Selectable, r.a.Principal.ID, in.SubmissionID, digest)
		return x, err
	})
}
