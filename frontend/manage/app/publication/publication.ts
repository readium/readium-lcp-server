import { CrudItem }  from '../crud/crud-item';

export class Publication implements CrudItem {
    id:             string | number;
    masterFilename: string;
    uuid:           string;
    title:          string;
    status:         string;
}
