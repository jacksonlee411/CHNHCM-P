import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Alert, Box, Button, MenuItem, Paper, Stack, TextField, Typography } from '@mui/material';
import { Link } from 'react-router-dom';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import {
  ObjectForm,
  ObjectTextField,
  ObjectDetailSection,
  RecordBackButton,
  RecordControl,
  UnsavedChangesDialog,
  recordedTime,
} from '@chnhcm/ui';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { useRuleSubmission } from './useRuleSubmission';
import { usePageTitle } from '../../shared/navigation';

type Period = { id: string; revision?: string; from: string; until?: string };
type View<R> = {
  name: string;
  version: string;
  rows: R[];
  canWrite: boolean;
  history: Models['DatedRuleSource'][];
};
type Props<R extends Period> = {
  title: string;
  back?: string;
  scope: string;
  resolve: (id: string) => Promise<Models['RuleSaved']>;
  session: Models['Principal'];
  load: (version: string, signal: AbortSignal) => Promise<View<R>>;
  save: (expected: string, row: R, key: string, insertAfter?: string) => Promise<Models['RuleSaved']>;
  blank: (from: string) => R;
  label: (row: R) => string;
  summary: (row: R) => ReactNode;
  fields: (row: R, update: (patch: Partial<R>) => void) => ReactNode;
};
export function DatedRules<R extends Period>({
  title,
  back,
  scope,
  resolve,
  session,
  load,
  save,
  blank,
  label,
  summary,
  fields,
}: Props<R>) {
  usePageTitle(title);
  const [version, setVersion] = useState(''),
    [refresh, setRefresh] = useState(0);
  const [view, setView] = useState<View<R>>(),
    [error, setError] = useState(''),
    [notice, setNotice] = useState(''),
    [stale, setStale] = useState(false);
  const [draft, setDraft] = useState<R>(),
    [original, setOriginal] = useState<R>(),
    [search, setSearch] = useState('');
  const anchor = useRef<string | undefined>(undefined),
    errorBox = useRef<HTMLDivElement>(null),
    returnFocus = useRef(false),
    newButton = useRef<HTMLButtonElement>(null),
    editButton = useRef('');
  const loadRef = useRef(load);
  loadRef.current = load;
  useEffect(() => {
    const controller = new AbortController(),
      same = sessionGuard();
    setView(undefined);
    setStale(false);
    loadRef
      .current(version, controller.signal)
      .then((v) => {
        if (!controller.signal.aborted && same()) setView(v);
      })
      .catch((e) => {
        if (!controller.signal.aborted && same()) setError(e.message);
      });
    return () => controller.abort();
  }, [version, refresh, session.subjectId, session.instanceId]);
  const close = () => {
    setDraft(undefined);
    setOriginal(undefined);
    returnFocus.current = true;
  };
  useEffect(() => {
    if (!draft && view && returnFocus.current) {
      (document.getElementById(editButton.current) ?? newButton.current)?.focus();
      returnFocus.current = false;
    }
  }, [draft, view]);
  useEffect(() => {
    if (error) errorBox.current?.focus();
  }, [error]);
  const submission = useRuleSubmission<{ expected: string; row: R; insertAfter?: string }>(scope, session, {
    send: (cmd, key) => save(cmd.expected, cmd.row, key, cmd.insertAfter),
    resolve,
    finished: (result, recovery) => {
      if (
        (result.outcome === 'accepted' || result.outcome === 'no_change') &&
        (!draft || recovery !== 'other')
      ) {
        close();
        setView(undefined);
        setRefresh((n) => n + 1);
      }
      setError('');
      setNotice(
        {
          accepted: recovery ? '原提交已保存。' : '已保存。',
          no_change: recovery ? '原提交未产生变更。' : '资料没有变化。',
          rejected: '原提交已拒绝，可核对后重新保存。',
          not_applied: '原提交未生效，可核对后重新保存。',
        }[result.outcome],
      );
    },
    failed: (e) => {
      if (e instanceof ApiError && [401, 403, 404].includes(e.status)) {
        close();
        setView(undefined);
        setRefresh((n) => n + 1);
      }
      setStale(
        e instanceof ApiError &&
          ['SOURCE_CHANGED', 'M02_SOURCE_CHANGED', 'M03_SOURCE_CHANGED', 'RULE_ANCHOR'].includes(e.code),
      );
      setError(e instanceof Error ? e.message : '暂时无法确认保存结果');
    },
  });
  const leave = useDraftExit(
    !!draft && JSON.stringify(draft) !== JSON.stringify(original),
    () => false,
    close,
  );
  const edit = (row: R, button: HTMLButtonElement, insertAfter?: string) => {
    anchor.current = insertAfter;
    editButton.current = button.id;
    setOriginal(row);
    setDraft(row);
    setError('');
    setStale(false);
    setNotice('');
    submission.start();
  };
  const update = (patch: Partial<R>) => {
    if (submission.busy) return;
    setDraft((r) => r && { ...r, ...patch });
    setError('');
  };
  const submit = async () => {
    if (!draft || !view || submission.busy) return;
    setError('');
    await submission.submit({ expected: view.version, row: draft, insertAfter: anchor.current });
  };
  return (
    <Box className="standard-page object-page" aria-busy={!view && !error}>
      <RecordControl
        navigation={null}
        identity={
          <>
            {back && <RecordBackButton component={Link} label="返回法律实体" to={back} />}
            <Typography component="h1" className="record-control-name">
              {title}
            </Typography>
          </>
        }
        actions={
          view?.canWrite &&
          !draft && (
            <Button
              ref={newButton}
              id="rule-new"
              disabled={submission.busy}
              variant="contained"
              size="small"
              onClick={(e) => edit(blank(view.rows.length ? '' : '1900-01-01'), e.currentTarget)}
            >
              新增
            </Button>
          )
        }
      />
      <Box className="page-scroll" data-page-scroll tabIndex={0} role="region" aria-label={title + '正文'}>
        <Stack spacing={2} sx={{ p: 2 }}>
          {submission.storageError && (
            <Alert severity="error">暂时无法读取上次保存进度，请刷新后重试。</Alert>
          )}
          {submission.pending.map((ref) => (
            <Alert
              key={ref.submissionId}
              severity="warning"
              action={<Button onClick={() => void submission.recover(ref)}>查看保存结果</Button>}
            >
              有一笔保存结果待确认。
            </Alert>
          ))}
          {error && (
            <Alert
              severity="error"
              tabIndex={-1}
              ref={errorBox}
              action={
                stale && draft ? (
                  <Button
                    onClick={() =>
                      leave.request(() => {
                        close();
                        setError('');
                        setRefresh((n) => n + 1);
                      })
                    }
                  >
                    重新读取当前资料
                  </Button>
                ) : (
                  !draft && (
                    <Button
                      onClick={() => {
                        setError('');
                        setRefresh((n) => n + 1);
                      }}
                    >
                      重试
                    </Button>
                  )
                )
              }
            >
              {error}
            </Alert>
          )}
          {notice && (
            <Alert role="status" severity="success">
              {notice}
            </Alert>
          )}
          {view && (
            <>
              <Typography variant="h6">{view.name}</Typography>
              {draft ? (
                <ObjectForm>
                  <form
                    onSubmit={(e) => {
                      e.preventDefault();
                      void submit();
                    }}
                  >
                    <Box
                      component="fieldset"
                      disabled={submission.busy}
                      sx={{ border: 0, p: 0, m: 0, minWidth: 0 }}
                    >
                      <ObjectDetailSection title={draft.revision ? '更正期间' : '新增期间'}>
                        {fields(draft, update)}
                        <ObjectTextField
                          type="date"
                          required
                          label="生效日期"
                          value={draft.from}
                          onChange={(e) => update({ from: e.target.value } as Partial<R>)}
                        />
                        <ObjectTextField
                          type="date"
                          label="首个不适用日"
                          value={draft.until ?? ''}
                          onChange={(e) => update({ until: e.target.value } as Partial<R>)}
                        />
                      </ObjectDetailSection>
                      <Stack direction="row" spacing={1}>
                        <Button type="submit" variant="contained" disabled={submission.busy}>
                          保存
                        </Button>
                        <Button disabled={submission.busy} onClick={() => leave.request(close)}>
                          取消
                        </Button>
                      </Stack>
                    </Box>
                  </form>
                </ObjectForm>
              ) : (
                <>
                  <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                    <TextField
                      size="small"
                      label="筛选规则"
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                    />
                    {view.history.length > 0 && (
                      <TextField
                        size="small"
                        select
                        label="资料版本"
                        value={version}
                        onChange={(e) => {
                          setNotice('');
                          setError('');
                          setVersion(e.target.value);
                        }}
                      >
                        <MenuItem value="">当前资料</MenuItem>
                        {view.history.map((h, i) => (
                          <MenuItem key={h.version} value={h.version}>
                            第 {view.history.length - i} 版 · {recordedTime(h.recordedAt)}
                          </MenuItem>
                        ))}
                      </TextField>
                    )}
                    {version && (
                      <Button
                        onClick={() => {
                          setError('');
                          setVersion('');
                        }}
                      >
                        返回当前资料
                      </Button>
                    )}
                  </Stack>
                  {view.rows
                    .filter((r) => label(r).includes(search.trim()))
                    .map((r) => (
                      <Paper key={r.id} variant="outlined" sx={{ p: 2, overflowWrap: 'anywhere' }}>
                        <Typography component="h2" variant="subtitle1">
                          {label(r)}
                        </Typography>
                        <Typography>
                          {r.from} 起{r.until ? '，' + r.until + ' 起不再适用' : ''}
                        </Typography>
                        {summary(r)}
                        {view.canWrite && (
                          <Button
                            id={'rule-edit-' + r.id}
                            disabled={submission.busy}
                            onClick={(e) => edit(r, e.currentTarget)}
                          >
                            更正期间
                          </Button>
                        )}
                        {view.canWrite && (
                          <Button
                            id={'rule-split-' + r.id}
                            disabled={submission.busy}
                            onClick={(e) =>
                              edit(
                                { ...r, id: crypto.randomUUID(), revision: undefined, from: '' },
                                e.currentTarget,
                                r.id,
                              )
                            }
                          >
                            插入后续期间
                          </Button>
                        )}
                      </Paper>
                    ))}
                  {!view.rows.some((r) => label(r).includes(search.trim())) && (
                    <Typography>没有匹配的规则。</Typography>
                  )}
                </>
              )}
            </>
          )}
        </Stack>
      </Box>
      <UnsavedChangesDialog {...leave.dialog} />
    </Box>
  );
}
