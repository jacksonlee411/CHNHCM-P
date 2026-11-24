import type { Models } from '@chnhcm/api-client';
import { ObjectDefinition } from './dated/presentation';
import { ObjectList, ObjectNew, ObjectDetail } from './DatedPages';
import { SubmissionRecovery } from './dated/SubmissionRecovery';
import { department } from './departmentAdapter';
export { DEPARTMENT_PATH } from './departmentAdapter';
export function DepartmentList() { return <ObjectDefinition value={department}><ObjectList /></ObjectDefinition>; }
export function NewDepartment({ session }: { session: Models['Principal'] }) { return <ObjectDefinition value={department}><ObjectNew session={session} /></ObjectDefinition>; }
export function DepartmentDetail({ session }: { session: Models['Principal'] }) { return <ObjectDefinition value={department}><ObjectDetail session={session} /></ObjectDefinition>; }
export function DepartmentRecovery({ session }: { session: Models['Principal'] }) { return <SubmissionRecovery definition={department} session={session} />; }
