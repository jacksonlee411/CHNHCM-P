import type { Models } from '@chnhcm/api-client';
import { ObjectDefinition } from './dated/presentation';
import { ObjectList, ObjectNew, ObjectDetail } from './DatedPages';
import { SubmissionRecovery } from './dated/SubmissionRecovery';
import { businessUnit } from './businessUnitAdapter';
export { BU_PATH } from './businessUnitAdapter';
export function BusinessUnitList() { return <ObjectDefinition value={businessUnit}><ObjectList /></ObjectDefinition>; }
export function NewBusinessUnit({ session }: { session: Models['Principal'] }) { return <ObjectDefinition value={businessUnit}><ObjectNew session={session} /></ObjectDefinition>; }
export function BusinessUnitDetail({ session }: { session: Models['Principal'] }) { return <ObjectDefinition value={businessUnit}><ObjectDetail session={session} /></ObjectDefinition>; }
export function Recovery({ session }: { session: Models['Principal'] }) { return <SubmissionRecovery definition={businessUnit} session={session} />; }
