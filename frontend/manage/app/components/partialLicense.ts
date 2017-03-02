import { User } from './user';

export const DEFAULT_PROFILE = 'http://readium.org/lcp/basic-profile';
export const LEGACY_PROFILE = 'http://readium.org/lcp/profile-1.0';

export const USERKEY_ALGO = 'http://www.w3.org/2001/04/xmlenc#sha256';
export const PROVIDER = 'http://edrlab.org';

export class Key {
    algorithm: string;
}

export class ContentKey extends Key {
    encrypted_value: any[] | undefined;
}

export class UserKey extends Key {
    text_hint:   string;
    key_check:   any[] | undefined;
    value:       any[] | undefined;
    clear_value: string | undefined;
}

export class  Encryption {
    profile:     string;
    content_key: ContentKey | undefined;
    user_key:    UserKey    | undefined;
}

export class Link {
    rel:       string;
    href:      string;
    type:      string  | undefined;
    title:     string  | undefined;
    profile:   string  | undefined;
    templated: boolean | undefined;
    size:      number  | undefined;
    checksum:  string  | undefined;
};

export class UserRights {
    print: number | undefined;
    copy:  number | undefined;
    start: Date |undefined;
    end:   Date |undefined;
}

export class  UserInfo  {
    id:        string;
    email:     string;
    name:      string;
    encrypted: string[] | undefined;
}


export class PartialLicense {
    provider:   string;  // 'http://edrlab.org'
    user:       UserInfo;  // get it from  user.user_id, user_email, ...
    encryption: Encryption; 
    rights:     UserRights | undefined;
}

export class PartialLicenseJSON extends PartialLicense {
    // function to encode / decode JSON string
    
}
