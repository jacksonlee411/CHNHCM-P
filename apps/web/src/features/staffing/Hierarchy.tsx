import {useState, useRef, useEffect, useLayoutEffect} from 'react';
import {Alert,Box,Button,IconButton,Stack,Typography} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import {Link,useSearchParams} from 'react-router-dom';
import {ObjectTextField,recordedTime,HistoryNotice,ListControl,RecordBackButton,SearchControl} from '@chnhcm/ui';
import {usePageTitle} from '../../shared/navigation';
import {paths} from './api';
import {useStructure,nodeHref,nodeLabel,structurePath,type Node,type Query} from './structure';

export {structurePath as hierarchyPath};
type Common = {day?:string; version?:string};
function Pager({page,more,move}:{page:number;more?:boolean;move:(p:number)=>void}) {
  if (page===1 && !more) return null;
  return <Stack direction="row" alignItems="center">
    <Button disabled={page===1} onClick={()=>move(page-1)}>上一页</Button>
    <Typography>{page}</Typography><Button disabled={!more} onClick={()=>move(page+1)}>下一页</Button>
  </Stack>
}
export function StructureControls({day,change}:{day:string;change:(day:string)=>void}) {
  const [input,setInput]=useState<string>();
  const [dayAtFocus,setDayAtFocus]=useState<string>();
  const text=input??dayAtFocus??day;
  return <Stack component="form" direction={{xs:'column',sm:'row'}} spacing={1} onSubmit={event=>{
    event.preventDefault();
    change(text);
    setInput(undefined);
    setDayAtFocus(undefined);
  }}>
    <ObjectTextField required type="date" label="结构日期" value={text}
      onFocus={e=>setDayAtFocus(e.target.value)} onBlur={()=>setDayAtFocus(undefined)}
      onChange={e=>setInput(e.target.value)}/>
    <Button type="submit">查询结构</Button>
  </Stack>
}
type BranchProps=Common & {nodeId?:string;search?:string;back:string;focus?:string;locate:(id:string)=>void; depth?:number;memory:Map<string,number|boolean>};
function Branch(props:BranchProps) {
  const {day,version,nodeId,search,back,memory}=props;
  const key='page:'+(nodeId??'')+':'+(search??'');
  const [page,setPage]=useState(Number(memory.get(key))||1),[retry,setRetry]=useState(0);
  const query:Query={mode:search?'search':'nodes',businessAt:day,sourceVersion:version,nodeId,q:search,page};
  const result=useStructure(query,true,retry);
  if (result?.error) return <Alert severity="error" action={<Button onClick={()=>setRetry(n=>n+1)}>重试</Button>}>{result.error}</Alert>;
  if (!result?.data) return <Typography role="status">正在读取…</Typography>;
  const data=result.data;
  return <Box>
    {data.nodes.length===0 && <Typography>{search?'没有匹配岗位。':'该日期没有可显示的岗位。'}</Typography>}
    <Box component="ul" onKeyDown={event=>{
      if(!['ArrowDown','ArrowUp','Home','End'].includes(event.key))return;
      const root=event.currentTarget.closest('[data-position-tree]')??event.currentTarget;
      const links=Array.from(root.querySelectorAll<HTMLAnchorElement>('li a[href]'));
      const index=links.indexOf(event.target as HTMLAnchorElement);
      if(index<0)return;
      const next=event.key==='Home'?0:event.key==='End'?links.length-1:Math.min(links.length-1,Math.max(0,index+(event.key==='ArrowUp'?-1:1)));
      event.preventDefault();event.stopPropagation();links[next]?.focus();
    }} aria-label={search?'岗位搜索结果':'岗位上下级关系'} sx={{listStyle:'none',m:0,p:0}}>
      {data.nodes.map(node=><TreeRow key={node.id} node={node} {...props} day={data.businessAt} back={back}/>)}
    </Box>
    <Pager page={page} more={data.more} move={n=>{memory.set(key,n);setPage(n)}}/>
  </Box>
}
function TreeRow({node,...props}:BranchProps & {node:Node}){
  const [open,setOpen]=useState(!!props.memory.get(node.id));
  const focus=useRef<HTMLAnchorElement>(null);
  useEffect(()=>{if(props.focus===node.id) focus.current?.focus()},[props.focus,node.id]);
  const expand=(v:boolean)=>{props.memory.set(node.id,v);setOpen(v)};
  const backParams=new URLSearchParams(props.back.split('?')[1]);backParams.set('focus',node.id);
  const href=nodeHref(node.id,props.day!,props.version??'',structurePath+'?'+backParams);
  return <Box component="li" sx={{py:.5}}>
    <Stack direction="row" alignItems="center" onKeyDown={e=>{
      if(e.key==='ArrowRight' && node.hasChildren){e.preventDefault();expand(true)}
      if(e.key==='ArrowLeft' && open){e.preventDefault();expand(false)}
    }}>
      {node.hasChildren && !props.search && <IconButton size="small" aria-label={(open?'收起':'展开')+node.name} aria-expanded={open} onClick={()=>expand(!open)}>{open?<ExpandMoreIcon/>:<ChevronRightIcon/>}</IconButton>}
      <Link className="object-name-link" ref={focus} to={href} style={{overflowWrap:'anywhere'}}>{nodeLabel(node)}</Link>
      {props.search && <Button onClick={()=>props.locate(node.id)}>定位</Button>}
    </Stack>
    {open && node.hasChildren && !props.search && <Box sx={{pl:(props.depth??0)<4?1.5:0}}><Branch {...props} depth={(props.depth??0)+1} nodeId={node.id}/></Box>}
  </Box>
}
function VersionPicker({day,select}:{day:string;select:(version:string)=>void}){
  const [page,setPage]=useState(1),[retry,setRetry]=useState(0);
  const result=useStructure({mode:'versions',businessAt:day||undefined,page},true,retry);
  if(result?.error)return <Alert severity="error" action={<Button onClick={()=>setRetry(n=>n+1)}>重试</Button>}>{result.error}</Alert>;
  return <Box aria-label="发布历史">{!result?.data ? <Typography>正在读取…</Typography> :
    <>{result.data.versions?.length ? result.data.versions.map(v=><Button key={v.sourceVersion} onClick={()=>select(v.sourceVersion)}>第 {v.number} 版 · {recordedTime(v.recordedAt)}</Button>) : <Typography>尚无发布记录。</Typography>}<Pager page={page} move={setPage} more={result.data.more}/></>}
  </Box>
}
// Per-tab return state contains only expansion and paging, never labels or grants.
const memories=new Map<string,Map<string,number|boolean>>();
export function HierarchyPage({session}:{session:{instanceId:string;subjectId:string}}){
  usePageTitle('岗位上下级');
  const [params,setParams]=useSearchParams();
  const latestParams = useRef(params);
  useLayoutEffect(() => { latestParams.current = params; }, [params]);
  const day=params.get('businessAt')??'',version=params.get('sourceVersion')??'';
  const search=params.get('q')??'',focus=params.get('focus')??'';
  const [input,setInput]=useState(search),[history,setHistory]=useState(false);
  useEffect(()=>setInput(search),[search]);
  useEffect(()=>{const clear=()=>memories.clear();window.addEventListener('chnhcm-session-invalidated',clear);return ()=>window.removeEventListener('chnhcm-session-invalidated',clear)},[]);
  const key=JSON.stringify([session.instanceId,session.subjectId,day,version,search]);
  if(!memories.has(key)){if(memories.size>=20)memories.delete(memories.keys().next().value!);memories.set(key,new Map())}
  const change=(values:Record<string,string>)=>{
    // A name search can follow a date submission before the router renders it.
    const next=new URLSearchParams(latestParams.current);
    for(const [key,value] of Object.entries(values)) if(value)next.set(key,value);else next.delete(key);
    latestParams.current=next;
    setParams(next,{replace:true,flushSync:true});setHistory(false)
  };
  const [retry,setRetry]=useState(0);
  const root=useStructure({mode:'nodes',businessAt:day||undefined,sourceVersion:version||undefined,pageSize:1},true,retry);
  const path=useStructure({mode:'path',businessAt:day||undefined,sourceVersion:version||undefined,nodeId:focus},!!focus,retry);
  const located=path?.data?.nodes;
  const range=memories.get(key)!;
  // A deep link can point to a sibling outside the first branch window.
  // The server supplies the page of each ancestor under the same dated sort.
  if(focus && !search && located){
    for(const node of located){
      range.set('page:'+(node.parentId??'')+':',node.siblingPage??1);
      if(node.id!==focus)range.set(node.id,true);
    }
  }
  const back=structurePath+'?'+params;
  const listBack = params.get('listBack') ?? paths.position;
  return <Box className="standard-page" aria-busy={!root}>
    <ListControl title="岗位上下级"
      action={<RecordBackButton component={Link} to={listBack.split('?')[0] === paths.position ? listBack : paths.position} />}
      search={<SearchControl value={input} onChange={setInput} label="查找岗位" onSearch={()=>change({q:input.trim(),focus:''})}/>}
    />
    {version && <HistoryNotice label="返回当前结构" onReturn={()=>change({sourceVersion:''})}/>}
    <Box className="page-scroll" data-page-scroll data-page-scroll-ready={!!root} tabIndex={0} aria-label="岗位上下级" sx={{p:2}}>
      <Stack spacing={1}>
        <StructureControls key={version}
          day={day||root?.data?.businessAt||''}
          change={date=>change({businessAt:date,focus:''})}/>
        {root?.error && <Alert severity="error" action={<Button onClick={()=>setRetry(n=>n+1)}>重试</Button>}>{root.error}</Alert>}
        {root?.data && <>
          {search && <Button sx={{alignSelf:'start'}} onClick={()=>change({q:''})}>清除搜索</Button>}
          {root.data.canReadHistory && <Button sx={{alignSelf:'start'}} onClick={()=>setHistory(!history)} aria-expanded={history}>发布历史</Button>}
          {history && root.data.canReadHistory && <VersionPicker day={day} select={source=>change({sourceVersion:source})}/>}
          {focus && (path?.error ? <Alert severity="warning">{path.error}</Alert> : <Stack direction="row" flexWrap="wrap" aria-label="岗位路径">
            {path?.data?.nodes.map(n=><Button component={Link} key={n.id} to={nodeHref(n.id,root.data!.businessAt,version,back)}>{nodeLabel(n)}</Button>)}
          </Stack>)}
          <Box data-position-tree>{(!focus || search || path) && <Branch key={key+':'+focus+':'+(located?.length??0)} day={root.data.businessAt} version={version||undefined} search={search||undefined} focus={focus}
            back={back} locate={id=>change({focus:id,q:''})} memory={range}/>}</Box>
        </>}
      </Stack>
    </Box>
  </Box>
}
