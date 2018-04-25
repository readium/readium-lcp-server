import { Component, OnInit }        from '@angular/core';
import { ActivatedRoute, Params }   from '@angular/router';
import 'rxjs/add/operator/switchMap';

import { User }                     from './user';
import { UserService }              from './user.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-user-edit',
    templateUrl: 'user-edit.component.html'
})

export class UserEditComponent implements OnInit {
    user: User;

    constructor(
        private route: ActivatedRoute,
        private userService: UserService) {
    }

    ngOnInit(): void {
        this.route.params
            .switchMap((params: Params) => this.userService.get(params['id']))
            .subscribe(user => {
                this.user = user
            });
    }
}
