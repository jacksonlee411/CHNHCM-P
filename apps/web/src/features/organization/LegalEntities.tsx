import type { Models } from '@chnhcm/api-client';
import { ObjectDefinition } from './dated/presentation';
import { ObjectList, ObjectDetail, ObjectNew } from './DatedPages';
import { SubmissionRecovery } from './dated/SubmissionRecovery';
import { legalEntity } from './legalEntityAdapter';
export { LEGAL_ENTITY_PATH } from './legalEntityAdapter';
export function LegalEntityList() { return <ObjectDefinition value={legalEntity}><ObjectList /></ObjectDefinition>; }
export function LegalEntityDetail({ session }: {
    session: Models['Principal'];
}) { return <ObjectDefinition value={legalEntity}><ObjectDetail session={session}/></ObjectDefinition>; }
export function LegalEntityRecovery({ session }: {
    session: Models['Principal'];
}) { return <SubmissionRecovery definition={legalEntity} session={session}/>; }
export function NewLegalEntity({ session }: {
    session: Models['Principal'];
}) { return <ObjectDefinition value={legalEntity}><ObjectNew session={session}/></ObjectDefinition>; }
