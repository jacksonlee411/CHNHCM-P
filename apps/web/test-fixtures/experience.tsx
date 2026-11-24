// Development-only entry, excluded from the production index.html build.
import { useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Box, Button, Tab, Tabs, Typography } from '@mui/material';
import { AppShell, EffectiveDates, ObjectDetailSheet, ObjectDetailIdentity, ObjectDetailSection, ObjectForm, ObjectFormError, ObjectFormSection, ObjectTextField, ObjectTabPanel, RecordControl, useObjectTabs, recordedTime } from '@chnhcm/ui';

function DateFixture() {
  const [date, setDate] = useState('2026-01-13'), [bounded, setBounded] = useState('2026-01-13');
  return <AppShell><Box className="standard-page"><Typography>日期组件夹具，非业务验收</Typography><ObjectForm>
    <ObjectTextField id="fixture-date" label="测试日期" type="date" required value={date} onChange={e => setDate(e.target.value)} />
    <ObjectTextField id="fixture-bounded-date" label="范围日期" type="date" value={bounded} onChange={e => setBounded(e.target.value)} slotProps={{ htmlInput: { min: '2026-01-10', max: '2026-01-20' } }} />
    <ObjectTextField label="只读日期" type="date" value="2026-01-13" slotProps={{ htmlInput: { readOnly: true } }} />
    <ObjectTextField label="禁用日期" type="date" value="2026-01-13" disabled />
    <Typography role="status">输入值：{date}</Typography>
    <Typography>{recordedTime('2026-01-12T16:30:05.123456Z')}</Typography>
  </ObjectForm></Box></AppShell>;
}

function Fixture() {
  const dates = Array.from({ length: 12 }, (_, index) => `2026-${String(index + 1).padStart(2, '0')}-01`);
  const [selected, setSelected] = useState(dates[5]);
  const [count, setCount] = useState(12);
  const visibleDates = count === 1 ? [dates[5]] : dates.slice(0, count);
  const [name, setName] = useState(''), [error, setError] = useState('');
  const tabs = useObjectTabs('basic', true, () => {});
  return <AppShell><Box className="standard-page object-page">
    <RecordControl identity={<Typography>组件展示夹具，非业务验收</Typography>} navigation={null} actions={<Button onClick={() => { setError('请核对明细名称'); tabs.focusField('fixture-name', 'fixture-errors'); }}>核对输入</Button>} />
    <Box className="page-scroll" data-page-scroll role="region" aria-label="夹具正文" tabIndex={0}>
      <ObjectDetailSheet identity={<ObjectDetailIdentity icon={<span>样</span>} title="共享体验组件" />} tabs={<Tabs value={tabs.tab} onChange={(_, value) => tabs.select(value)}><Tab id="fixture-basic-tab" value="basic" label="基本资料" aria-controls="fixture-basic-panel" /><Tab id="fixture-extra-tab" value="extra" label="补充资料" aria-controls="fixture-extra-panel" /></Tabs>} timeline={<EffectiveDates dates={visibleDates} selected={selected} onSelect={setSelected} />}>
        <Typography role="status">选中记录：{selected}</Typography><Button onClick={() => { setCount(1); setSelected(dates[5]); }}>单记录</Button><Button onClick={() => { setCount(0); setSelected(''); }}>空集合</Button>
        <ObjectForm><ObjectFormError id="fixture-errors" message={error} issues={error ? [{ id: 'fixture-name', message: error }] : []} onLocate={id => tabs.focusField(id, 'fixture-errors')} />
          <ObjectTabPanel prefix="fixture" value="basic" selected={tabs.tab}><ObjectDetailSection title="基本资料"><Typography>日期选择只改变展示定位。</Typography></ObjectDetailSection></ObjectTabPanel>
          <ObjectTabPanel prefix="fixture" value="extra" selected={tabs.tab}><ObjectFormSection title="折叠明细"><ObjectTextField id="fixture-name" label="明细名称" value={name} onChange={event => setName(event.target.value)} /></ObjectFormSection></ObjectTabPanel>
        </ObjectForm>
      </ObjectDetailSheet>
    </Box>
  </Box></AppShell>;
}
createRoot(document.getElementById('root')!).render(new URLSearchParams(location.search).has('date-fields') ? <DateFixture /> : <Fixture />);
