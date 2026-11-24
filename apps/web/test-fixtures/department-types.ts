// Public object contracts remain distinct even though their presentation is shared.
import type {Models} from '@chnhcm/api-client';
function ports(department: Models['DepartmentCreateInput'], bu: Models['CreateInput'], result: Models['DepartmentResult']) {
  // @ts-expect-error BU create cannot supply the mandatory department relationship.
  const missingBU: Models['DepartmentCreateInput'] = bu;
  // @ts-expect-error Department has no BU result identity.
  void result.buId;
  // @ts-expect-error A department patch does not accept the BU English-name field.
  const wrongField: Models['DepartmentChangesInput'] = { nameEn: 'English' };
  // @ts-expect-error A BU patch does not accept department relationships.
  const wrongBU: Models['ChangesInput'] = { managementBuId: department.managementBuId };
  void missingBU; void wrongField; void wrongBU;
}
void ports;
