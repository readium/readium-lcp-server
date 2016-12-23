import { Component, OnInit } from '@angular/core';
import { User } from './user';
import { UserService } from './user.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-user-list',
    templateUrl: 'user-list.component.html'
})

export class UserListComponent implements OnInit {
    users: User[];

    constructor(private userService: UserService) {
        this.users = [];
    }

    refreshUsers(): void {
        this.userService.getUsers().then(
            users => {
                this.users = users;
            }
        );
    }

    ngOnInit(): void {
        this.refreshUsers();
    }

    onRemove(objId: any): void {
        this.userService.delete(objId).then(
            user => {
                this.refreshUsers();
            }
        );
    }
 }
