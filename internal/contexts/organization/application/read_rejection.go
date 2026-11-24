package application

import "chnhcm.local/chnhcm/internal/contexts/organization/application/submission"

type ReadRejection = submission.ReadRejection

func RejectRead(err error) error { return submission.RejectRead(err) }
