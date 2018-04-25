import { CrudItem }  from '../crud/crud-item';

export class User implements CrudItem {
    id:             number;
    uuid:           string;
    name:           string;
    email:          string;
    password?:      string | undefined | null; // Hashed password
    clearPassword?: string | undefined | null;
    hint?:          string | undefined | null;
}
