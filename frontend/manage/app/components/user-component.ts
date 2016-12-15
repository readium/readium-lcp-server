import { Component, Input, OnInit } from '@angular/core';
import {ActivatedRoute, Params} from '@angular/router';
import {Location} from '@angular/common';

import { User } from './user';
import { UserService } from './user.service';

@Component({
    moduleId: module.id,
    selector: 'user',
    templateUrl: '/app/components/user.html',
    styleUrls: ['../../app/components/user.css'],
    providers: [UserService]
})

export class UserComponent implements OnInit {
    @Input() user: User;

    constructor(
        private userService: UserService,
        private route: ActivatedRoute,
        private location: Location
    ) {}

    ngOnInit(): void {
        this.route.params.forEach((params: Params) => {
            let id = +params['id'];
            this.userService.getUser(id)
            .then(user => this.user = user);
        });
    }
    goBack(): void {
        this.location.back();
    }

    save(): void {
        this.userService.update(this.user)
        .then(() => this.goBack());
    }
}
