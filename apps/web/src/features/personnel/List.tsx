import {Alert, Button} from '@mui/material';
import {Link, useLocation, useSearchParams} from 'react-router-dom';
import {ListControl} from '@chnhcm/ui';
import {EmployeeDirectory} from '../hire/EmployeeDirectory';
import {usePageTitle} from '../../shared/navigation';
import * as person from './api';

export function PersonList({session}:{session:person.Principal}) {
  usePageTitle('人员');
  const route=useLocation(), [params]=useSearchParams();
  const access=person.useRead(session.subjectId,person.entry);
  const register=access.data?.canRegister && <Button variant="contained" component={Link}
    to={person.path+'/new?'+person.directoryParams(params)}>登记人员</Button>;
  if(access.data?.canRead) return <EmployeeDirectory session={session} register={register}/>;
  return <div className="standard-page"><ListControl title="人员" action={register}>
    {access.error && <Alert severity="error" action={<Button onClick={access.refresh}>重试</Button>}>{access.error}</Alert>}
    {access.data && !register && '没有人员名录权限'}
    {route.state?.registered && register && <Alert severity="success">人员登记已保存</Alert>}
  </ListControl></div>;
}
