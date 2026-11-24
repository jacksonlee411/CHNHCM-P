package application

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

const hierarchyKind = d.Kind(s.Hierarchy)

func hierarchyReader(ctx context.Context, u Unit, ent, date string, exact bool) ([]string, error) {
	fields, err := decide(ctx, u, d.Position, ent, "read", nil)
	if err != nil {
		return nil, err
	}
	if date != today(u) || exact {
		history, err := decide(ctx, u, d.Position, ent, "history.read", nil)
		if err != nil {
			return nil, err
		}
		fields.ReadFields = intersection(fields.ReadFields, history.ReadFields)
	}
	if !has(fields.ReadFields, "name", "effectiveDate", "status") {
		return nil, foundation.Fail(403, "M03_FORBIDDEN", "没有读取岗位名称及有效期间的权限")
	}
	return fields.ReadFields, nil
}

func (svc Service) HierarchyAt(ctx context.Context, cred foundation.Credentials, date, version string) (out s.HierarchyView, err error) {
	if date != "" && !foundation.ValidDate(date) || version != "" && !foundation.ValidID(version) {
		return out, foundation.Fail(400, "M03_INVALID_INPUT", "业务日期或发布版本无效")
	}
	err = svc.Store.Within(ctx, cred, false, func(u Unit) error {
		ent, err := u.Enterprise(ctx, false)
		if err != nil {
			return err
		}
		if _, err = decide(ctx, u, hierarchyKind, ent.ID, "read", nil); err != nil {
			return err
		}
		if date == "" {
			date = today(u)
		}
		fields, err := hierarchyReader(ctx, u, ent.ID, date, version != "")
		if err != nil {
			return err
		}
		permit := func(action string) (bool, error) {
			decision, err := u.Access().Decide(ctx, access.Intent{Action: string(s.Hierarchy) + "." + action, Kind: string(s.Hierarchy), EnterpriseID: ent.ID})
			return decision.Allowed, err
		}
		history, err := permit("history.read")
		if err != nil {
			return err
		}
		if version != "" && !history {
			return foundation.Fail(403, "M03_FORBIDDEN", "无权读取历史结构")
		}
		publish, err := permit("publish")
		if err != nil {
			return err
		}
		root, versions, err := u.HierarchyVersions(ctx, ent.ID)
		if err != nil {
			return err
		}
		selected := version
		if version != "" {
			if !slices.ContainsFunc(versions, func(v s.HierarchyVersion) bool { return v.SourceVersion == version }) {
				return foundation.Fail(404, "M03_NOT_FOUND", "找不到指定的管理汇报版本")
			}
		} else if len(versions) > 0 {
			selected = versions[0].SourceVersion
		}
		edges, err := u.HierarchyEdges(ctx, root, selected)
		if err != nil {
			return err
		}
		heads, err := u.HierarchyPositions(ctx, ent.ID, date, root, version)
		if err != nil {
			return err
		}
		nodes := map[string]d.Timeline{}
		for _, position := range heads {
			if at := d.At(position, date); at >= 0 && position.Records[at].Active() {
				nodes[position.ID] = position
			}
		}
		parents := edges.At(date)
		out = s.HierarchyView{HierarchyID: root, BusinessAt: date, SourceVersion: selected, Nodes: []s.HierarchyNode{}, Versions: []s.HierarchyVersion{}, CanPublish: publish && version == ""}
		if history {
			out.Versions = versions
		}
		for _, position := range nodes {
			index := d.At(position, date)
			if index < 0 || !position.Records[index].Active() {
				return errors.New("missing effective published position")
			}
			node := s.HierarchyNode{ID: position.ID, Name: position.Records[index].Data["name"], ParentID: parents[position.ID]}
			if has(fields, "code") {
				node.Code = position.Code
			}
			out.Nodes = append(out.Nodes, node)
		}
		for child, parent := range parents {
			if _, ok := nodes[child]; !ok {
				return errors.New("missing hierarchy child")
			}
			if _, ok := nodes[parent]; !ok {
				return errors.New("missing hierarchy parent")
			}
		}
		slices.SortFunc(out.Nodes, func(left, right s.HierarchyNode) int {
			return cmp.Or(cmp.Compare(left.Name, right.Name), cmp.Compare(left.ID, right.ID))
		})
		return nil
	})
	return
}

func (svc Service) PublishHierarchy(ctx context.Context, cred foundation.Credentials, key string, digest []byte, input s.PublishHierarchy) (out s.Result, err error) {
	if !foundation.ValidID(key) || len(digest) != 32 {
		return out, foundation.Fail(400, "M03_INVALID_INPUT", "提交标识无效")
	}
	err = svc.Store.Within(ctx, cred, true, func(u Unit) error {
		ent, err := u.Enterprise(ctx, true)
		if err != nil {
			return err
		}
		if _, err = decide(ctx, u, hierarchyKind, ent.ID, "read", nil); err != nil {
			return err
		}
		if _, err = decide(ctx, u, hierarchyKind, ent.ID, "publish", nil); err != nil {
			return err
		}
		if err = u.LockSubmission(ctx, hierarchyKind, key); err != nil {
			return err
		}
		old, err := u.Terminal(ctx, hierarchyKind, key)
		if err != nil {
			return err
		}
		if old.Exists {
			if old.Result.Outcome != "not_applied" && !bytes.Equal(old.Digest, digest) {
				return foundation.Fail(409, "M03_SUBMISSION_MISMATCH", "提交标识已用于其他内容")
			}
			out = old.Result
			out.Replayed = true
			return nil
		}
		out = s.Result{Kind: s.Hierarchy, Action: "PUBLISH", SubmissionID: key}
		err = u.Business(ctx, func() error {
			if !foundation.ValidDate(input.EffectiveDate) || !foundation.ValidID(input.ChildID) || input.ParentID != nil && !foundation.ValidID(*input.ParentID) || input.ExpectedVersion != "" && !foundation.ValidID(input.ExpectedVersion) {
				return foundation.Fail(400, "M03_INVALID_INPUT", "请选择有效的生效日期和岗位")
			}
			if _, err := hierarchyReader(ctx, u, ent.ID, input.EffectiveDate, false); err != nil {
				return err
			}
			if err := u.LockHierarchy(ctx, ent.ID); err != nil {
				return err
			}
			root, versions, err := u.HierarchyVersions(ctx, ent.ID)
			if err != nil {
				return err
			}
			head, number := "", 1
			if len(versions) > 0 {
				head, number = versions[0].SourceVersion, versions[0].Number+1
			}
			if head != input.ExpectedVersion {
				return foundation.Fail(409, "M03_VERSION_CONFLICT", "岗位结构已发布新版本，请重新查询")
			}
			edges, err := u.HierarchyEdges(ctx, root, head)
			if err != nil {
				return err
			}
			child, parent := strings.ToLower(input.ChildID), ""
			if input.ParentID != nil {
				parent = strings.ToLower(*input.ParentID)
			}
			ids := []string{child}
			if parent != "" {
				ids = append(ids, parent)
			}
			for _, edge := range edges {
				ids = append(ids, edge.Child)
				if edge.Parent != "" {
					ids = append(ids, edge.Parent)
				}
			}
			slices.Sort(ids)
			ids = slices.Compact(ids)
			if err = u.LockHierarchyNodes(ctx, ids); err != nil {
				return err
			}
			positions, err := u.Heads(ctx, d.Position, ids)
			if err != nil {
				return err
			}
			byID := map[string]d.Timeline{}
			for _, position := range positions {
				byID[position.ID] = position
			}
			if index := d.At(byID[child], input.EffectiveDate); parent == "" && index < 0 {
				return foundation.Fail(409, "M03_HIERARCHY_CONFLICT", input.EffectiveDate+"：子岗位不存在或尚未生效")
			}
			if edges.At(input.EffectiveDate)[child] == parent {
				out.Outcome = "no_change"
				out.ObjectID = root
				out.SourceVersion = head
				return nil
			}
			next := edges.Assign(child, parent, input.EffectiveDate)
			if err = next.Validate(byID); err != nil {
				return err
			}
			if root == "" {
				root = foundation.NewID()
			}
			out.ObjectID = root
			out.SourceVersion = foundation.NewID()
			out.Outcome = "accepted"
			actor, err := u.Access().OperatorSnapshot(ctx)
			if err != nil {
				return err
			}
			return u.SaveHierarchy(ctx, ent.ID, out, number, next, actor)
		})
		if err != nil {
			var fault *foundation.Fault
			if !errors.As(err, &fault) || fault.Status >= 500 {
				return err
			}
			out = s.Result{Kind: s.Hierarchy, Action: "PUBLISH", SubmissionID: key, Outcome: "rejected", Failure: fault, FailureStatus: fault.Status}
		}
		return u.Close(ctx, hierarchyKind, key, digest, out)
	})
	return
}

func checkHierarchyUses(ctx context.Context, u Unit, plan d.Plan) error {
	if plan.After.Kind != d.Position || plan.Before.ID == "" || !d.CoverageChanged(plan.Before, plan.After) {
		return nil
	}
	edges, err := u.CurrentHierarchyUses(ctx, plan.After.Enterprise, plan.After.ID)
	if err != nil {
		return err
	}
	first := ""
	for _, edge := range edges {
		if date := d.PositionConflict(plan.After, edge.Start, edge.End, ""); date != "" && (first == "" || date < first) {
			first = date
		}
	}
	if first != "" {
		return foundation.Fail(409, "M03_HIERARCHY_CONFLICT", first+"：岗位仍参与已发布的上下级关系，请先调整结构")
	}
	return nil
}

func (svc Service) Structure(ctx context.Context, cred foundation.Credentials, q s.StructureQuery) (out s.StructureView, err error) {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 30
	}
	q.Q = strings.TrimSpace(q.Q)
	if q.BusinessAt != "" && !foundation.ValidDate(q.BusinessAt) ||
		q.SourceVersion != "" && !foundation.ValidID(q.SourceVersion) || q.NodeID != "" && !foundation.ValidID(q.NodeID) ||
		q.Page < 1 || q.Page > 1000000 || q.PageSize < 1 || q.PageSize > 100 || len([]rune(q.Q)) > 200 {
		return out, foundation.Fail(400, "M03_INVALID_INPUT", "岗位结构查询条件无效")
	}
	switch q.Mode {
	case "nodes", "versions", "candidates":
	case "search":
		if q.Q == "" {
			return out, foundation.Fail(400, "M03_INVALID_INPUT", "请输入岗位名称或编码")
		}
	case "path", "relation", "node":
		if q.NodeID == "" {
			return out, foundation.Fail(400, "M03_INVALID_INPUT", "请选择岗位")
		}
	default:
		return out, foundation.Fail(400, "M03_INVALID_INPUT", "岗位结构查询方式无效")
	}
	err = svc.Store.Within(ctx, cred, false, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		if _, e = decide(ctx, u, hierarchyKind, ent.ID, "read", nil); e != nil {
			return e
		}
		if q.BusinessAt == "" {
			q.BusinessAt = today(u)
		}
		fields, e := hierarchyReader(ctx, u, ent.ID, q.BusinessAt, q.SourceVersion != "" || q.Mode == "versions")
		if e != nil {
			return e
		}
		history, e := allowed(ctx, u, hierarchyKind, ent.ID, "history.read")
		if e != nil {
			return e
		}
		if history.Allowed {
			positions, err := allowed(ctx, u, d.Position, ent.ID, "history.read")
			if err != nil {
				return err
			}
			history.Allowed = positions.Allowed && has(positions.ReadFields, "name", "effectiveDate", "status")
		}
		if (q.Mode == "versions" || q.SourceVersion != "") && !history.Allowed {
			return foundation.Fail(403, "M03_FORBIDDEN", "无权读取历史结构")
		}
		pub, e := allowed(ctx, u, hierarchyKind, ent.ID, "publish")
		if e != nil {
			return e
		}
		out, e = u.ReadStructure(ctx, ent.ID, q, has(fields, "code"), history.Allowed)
		out.CanPublish = pub.Allowed && q.SourceVersion == ""
		return e
	})
	return
}
