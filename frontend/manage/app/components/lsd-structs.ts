
export class Updated {
    license: Date;
    status:  Date;
}

export class Link {
    rel: string;
    href: string;
    type: string;
    title: string;
    profile: string;
    templated: boolean;
}

export class PotentialRights {
    end: Date;
}


export class Event  {
    name: string; // device name
    timestamp: Date;
    type: string;
    id: string; // device ID
}

export class LicenseStatus {
    id:      string;
    status:          string;
    updated:         Updated;
    message:         string;
    links:           Link[];
    device_count:    number;
    potential_rights: PotentialRights;
    events:          Event[];
}
