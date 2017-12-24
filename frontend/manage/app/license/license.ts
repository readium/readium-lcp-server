import { Publication } from '../publication/publication';
import { User } from '../user/user';

export class License{
	id : string
	publicationTitle: string
	userName : string
    type: string
	devices: number
	status: string
	purchaseID: string
}