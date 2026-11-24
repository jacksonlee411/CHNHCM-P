import { useEffect, useState } from 'react';
import { Button, Stack, Typography } from '@mui/material';
import { Link, useLocation } from 'react-router-dom';
import type { ListFilterContext } from './dated/presentation';
import { api, ApiError, unwrap } from '@chnhcm/api-client';
export const geoLevels = [
    {value:'province', label:'省级'}, {value:'prefecture', label:'地市级'}, {value:'county', label:'区县级'},
    {value:'region', label:'区域(香港)'}, {value:'district', label:'地方行政区(香港)'},
    {value:'postal-region', label:'区域(澳门)'}, {value:'parish', label:'堂区(澳门)'},
    {value:'county-city', label:'县市(台湾)'}, {value:'township', label:'乡镇市区(台湾)'},
];
export const levelLabel = (level?: string | null) => geoLevels.find(item => item.value === level)?.label ?? '未提供';

// Each frame is a flat query. Repeated drill-downs never nest encoded return URLs.
export function childQuery(source: URLSearchParams, id: string) {
    const next = new URLSearchParams(source);
    const frame = new URLSearchParams(source);
    frame.delete('fromList');
    frame.set('selected', id);
    next.append('fromList', frame.toString());
    if (source.get('filterField') !== 'countryRegionId') {
        for (const key of ['filterField', 'filterOperator', 'filterValue']) next.delete(key);
    }
    for (const key of ['page', 'q', 'level', 'selected'])
        next.delete(key);
    next.set('parentAreaId', id);
    return next;
}

function previousQuery(query: URLSearchParams) {
    const frames = query.getAll('fromList');
    const last = frames.pop();
    if (!last) return undefined;
    const previous = new URLSearchParams(last);
    previous.delete('fromList');
    for (const frame of frames) previous.append('fromList', frame);
    return '?' + previous;
}

export function GeoAreaScope({ query, fields }: ListFilterContext) {
    const location = useLocation();
    const [parent, setParent] = useState<{ key: string; scope: string }>();
    const parentID = query.get('parentAreaId') ?? '';
    const canReadParent = fields.includes('parentAreaId');
    useEffect(() => {
        const controller = new AbortController();
        setParent(undefined);
        if (parentID && canReadParent) {
            void api.GET('/api/v1/geo-areas/{id}', { params: { path: { id: parentID } }, signal: controller.signal })
                .then(unwrap).then(result => {
                    if (!controller.signal.aborted) setParent({ key: location.key, scope: result.selectedRecord?.name ? result.selectedRecord.name + '的直接下级' : '直接下级（上级名称不可读）' });
                }).catch(error => {
                    if (!controller.signal.aborted) setParent({ key: location.key, scope: error instanceof ApiError && error.status === 403 ? '直接下级（上级名称不可读）' : '直接下级（上级名称暂不可用）' });
                });
        }
        return () => controller.abort();
    }, [parentID, canReadParent, location.key, fields.join('|')]);

    if (!parentID) return null;
    const clearScope = new URLSearchParams(query);
    for (const key of ['parentAreaId', 'page', 'selected', 'fromList']) clearScope.delete(key);
    const previous = previousQuery(query);
    return <Stack direction="row" useFlexGap flexWrap="wrap" gap={1} alignItems="center">
        <Typography variant="body2" role="status">{!canReadParent ? '直接下级范围不可用' : parent?.key === location.key ? parent.scope : '正在读取直接上级…'}</Typography>
        {previous && <Button component={Link} size="small" to={previous} state={{ restoreChildLink: true }}>返回上次列表</Button>}
        <Button component={Link} size="small" to={'?' + clearScope}>清除上级范围</Button>
    </Stack>;
}
