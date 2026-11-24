"""Current product protocol; one sample includes every required HTTP step."""
import json
import time
from urllib.parse import urlencode

def operation(self, role, name, client):
    route = '/api/v1/departments'
    steps=[]
    began=time.monotonic_ns()
    value=None
    path_population=None
    def call(method,path,body=None):
        step={'method':method,'entry':path.split('?')[0]};steps.append(step)
        try:
            value,ms=client.call(method,path,body)
            step.update(ok=True,status=200,ms=ms,request=getattr(client,'request_id',None))
            return value
        except Exception as error:
            step.update(ok=False,request=getattr(client,'request_id',None),status=getattr(error,'status',None),code=getattr(error,'code',None),ms=getattr(error,'elapsed_ms',None))
            raise
    def selection(value,page,selected):
        self.page(value['page'],self.ordered,page)
        summary=value['summary']
        assert summary['queryKey']==value['page']['queryKey']
        assert summary['selectedCount']==selected and summary['canExport']==(role=='admin')
        assert len(summary['draftId'])==36 and summary['revision']>=1
        assert 'selectionVersion' not in summary and 'totalCount' not in summary
    try:
        if name in ['list','path']:
            term=self.ordered[0]['path'].split(' / ')[0]
            rows=self.ordered if name=='list' else [r for r in self.ordered if term.lower() in r['path'].lower()]
            if name=='path': path_population=len(rows)
            query={} if name=='list' else dict(filterField='departmentPath',filterOperator='contains',filterValue=term)
            value=call('GET',route+'?'+urlencode(query))
            def check():
                if name=='list': self.page(value,rows)
                else:
                    assert 'count' not in value and value['hasMore']==(len(rows)>20) and bool(value.get('nextCursor'))==value['hasMore']
                    self.page(dict(value,count=len(rows)),rows)
        elif name.startswith('parent-'):
            _,mode,variant=name.split('-');searched=variant=='search'
            rows=[] if role=='reader' and mode=='create' else self.parent_expected[mode,searched]
            context=dict(mode='create',effectiveDate=self.day,managementBuId=self.parent_bu) if mode=='create' else self.context
            body=dict(context=context,q=self.search if searched else '',pageSize=20)
            value=call('POST',route+'/references/parents',body)
            def parent_page(result,index):
                assert result['applicable'] and 'count' not in result and 'page' not in result
                assert result['items']==rows[index*20:(index+1)*20]
                assert result['hasMore']==(len(rows)>(index+1)*20)
                assert bool(result.get('nextCursor'))==result['hasMore']
            if variant=='second' and value['hasMore']:
                parent_page(value,0)
                body['cursor']=value['nextCursor']
                value=call('POST',route+'/references/parents',body)
            def check(): parent_page(value,1 if variant=='second' else 0)
        else:
            explicit=name=='start-explicit'
            choice=dict(mode='explicit',ids=[self.ordered[0]['id'],self.ordered[-1]['id']]) if explicit else dict(mode='allMatching',excludedIds=[])
            value=call('POST',route+'/selection/start',dict(query={},selection=choice,page=1,pageSize=20,sort='code'))
            if name=='resolve':
                selection(value,1,len(self.rows));draft=value['summary']
                value=call('POST',route+'/selection/resolve',dict(query={},draftId=draft['draftId'],expectedRevision=draft['revision'],page=2,pageSize=20,sort='code'))
            def check(): selection(value,2 if name=='resolve' else 1,2 if explicit else len(self.rows))
        ended=time.monotonic_ns()
        sample=dict(ok=True,ms=(ended-began)/1e6)
        try: check()
        except Exception as error:sample.update(ok=False,semanticError=True,errorType=type(error).__name__)
    except Exception as error:
        ended=time.monotonic_ns()
        sample=dict(ok=False,ms=(ended-began)/1e6,httpError=not isinstance(error,(AssertionError,json.JSONDecodeError)),semanticError=isinstance(error,(AssertionError,json.JSONDecodeError)),status=getattr(error,'status',None),code=getattr(error,'code',None),errorType=type(error).__name__)
    sample.update(actualCallStartedNS=began,actualCallFinishedNS=ended,httpRequests=len(steps),subrequests=steps,request=steps[-1].get('request') if steps else None,requestIDs=[s['request'] for s in steps if s.get('request')])
    if path_population is not None:
        sample.update(pathQueryPolicy='first-department-root-name-after-306',expectedMatchingObjects=path_population,backgroundObjects=len(self.ordered))
    return sample
