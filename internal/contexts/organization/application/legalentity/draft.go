package legalentity

import shared "chnhcm.local/chnhcm/internal/contexts/organization/application/selection"

const DraftMaxMembers = shared.DraftMaxMembers
const DraftEditLimit = shared.DraftEditLimit

type Draft = shared.Draft
type DraftPageSelection = shared.DraftPageSelection
type DraftEdit = shared.DraftEdit
type Drafts = shared.Drafts

var DraftUnavailable = shared.DraftUnavailable
var DraftConflict = shared.DraftConflict
var DraftLimit = shared.DraftLimit
var DraftMemberLimit = shared.DraftMemberLimit
var DraftBatchLimit = shared.DraftBatchLimit
