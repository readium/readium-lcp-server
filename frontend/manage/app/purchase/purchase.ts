import { CrudItem }  from '../crud/crud-item';
import { Publication }  from '../publication/publication';
import { User }  from '../user/user';

export class Purchase implements CrudItem {
    id:                 number;
    uuid:               string;
    publication:        Publication;
    user:               User;
    type:               string;
    endDate:            string;
    transactionDate?:   string;
    startDate?:         string;
    licenseUuid?:       string | null;
    status?:            string;
    maxEndDate?:        string;
}
