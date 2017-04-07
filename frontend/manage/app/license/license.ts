import { Publication } from '../publication/publication';
import { User } from '../user/user';

export class License{
	id : string
	publication: Publication
	user : User
    type: string
	devices: number
	status: string
	message: string
}