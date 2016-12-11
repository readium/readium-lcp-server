import { User } from './user';

export class Purchase {
    user:            User;
    purchaseID:      Number;
    resource:        String;
    label:           string;
    licenseID:       string;
    transactionDate: Date;
    partialLicense:  string;
}
