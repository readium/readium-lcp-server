import { CrudItem }  from '../crud/crud-item';

export class Publication implements CrudItem {
    id:              number;
    uuid:            string;
    title:           string;
    masterFilename?: string;
    status?:         string;
}
