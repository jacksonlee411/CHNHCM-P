// Application return-hook checks against a delayed rendering adapter.
// This does not mount DataGrid or prove its virtualization behavior.
import { useEffect, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { createBrowserRouter, Link, RouterProvider, useSearchParams } from 'react-router-dom';
import { AppShell } from '@chnhcm/ui';
import type { GridApi } from '@mui/x-data-grid';
import { useListReturn } from '../src/shared/navigation';

const listPath = '/test-fixtures/list-return.html';
function Fixture() {
  const [params] = useSearchParams();
  const selected = params.get('selected');
  const [ready, setReady] = useState(false), [visible, setVisible] = useState(false);
  const rows = Array.from({ length: 20 }, (_, index) => ({ id: index === 7 ? 'target' : `row-${index}` }));
  const pending = useRef(false), renderTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const requests = useRef<number[]>([]);
  const apiRef = useRef<GridApi | null>(null);
  const { scroller, remember } = useListReturn({ key: 'list-return-ui-fixture', ready, apiRef, selected, rows, pageOffset: 40 });
  const adapter: Pick<GridApi, 'scroll' | 'scrollToIndexes' | 'getScrollPosition' | 'getRowElement' | 'subscribeEvent'> = {
    // This synthetic renderer has no grid measurement events; actual DataGrid is checked separately.
    subscribeEvent: () => () => {},
    scroll: ({ top }) => { const node = scroller.current?.querySelector('.MuiDataGrid-virtualScroller'); if (node && top !== undefined) node.scrollTop = top; },
    scrollToIndexes: ({ rowIndex }) => {
      requests.current.push(rowIndex!);
      if (!pending.current) {
        pending.current = true;
        renderTimer.current = setTimeout(() => setVisible(true), params.get('slowRow') === '1' ? 1400 : 60);
      }
      return true;
    },
    getScrollPosition: () => ({ top: scroller.current?.querySelector('.MuiDataGrid-virtualScroller')?.scrollTop ?? 0, left: 0 }),
    getRowElement: id => id === 'target' ? scroller.current?.querySelector<HTMLDivElement>('[data-fixture-target]') ?? null : null,
  };
  apiRef.current = adapter as GridApi;
  useEffect(() => {
    const timer = setTimeout(() => setReady(true), 250);
    return () => { clearTimeout(timer); if (renderTimer.current) clearTimeout(renderTimer.current); };
  }, []);
  return <AppShell><div className="standard-page">
    <h1 tabIndex={-1} data-list-heading>返回适配夹具</h1>
    <input aria-label="等待期间输入" />
    {ready && selected === 'missing' && <div tabIndex={-1} data-list-return-fallback>目标不在当前授权页</div>}
    <div ref={scroller} data-fixture-ready={ready} data-fixture-indexes={requests.current.join(',')}>
      <div className="MuiDataGrid-virtualScroller" style={{ height: 260, overflowY: 'auto' }}>
        <div style={{ height: 500 }}>延迟渲染的合成页面第三页</div>
        {visible && <div data-fixture-target><Link data-object-link tabIndex={-1} onClick={remember} to="/test-fixtures/return-detail">目标对象</Link></div>}
        <div style={{ height: 500 }} />
      </div>
    </div>
  </div></AppShell>;
}
function Detail() { return <AppShell><Link to={`${listPath}?selected=target`}>返回列表</Link></AppShell>; }
createRoot(document.getElementById('root')!).render(<RouterProvider router={createBrowserRouter([
  { path: listPath, element: <Fixture /> }, { path: '/test-fixtures/return-detail', element: <Detail /> },
])} />);
