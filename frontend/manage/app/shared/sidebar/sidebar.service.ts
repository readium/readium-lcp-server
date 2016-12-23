import { Injectable } from '@angular/core';
import { Subject }    from 'rxjs/Subject';

@Injectable()
export class SidebarService {
    private openSource = new Subject<boolean>();
    private open: boolean = false;

    // Observable string streams
    open$ = this.openSource.asObservable();

    toggle() {
        this.open = !(this.open);
        this.openSource.next(this.open);
    }
}
