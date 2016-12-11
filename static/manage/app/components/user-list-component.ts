import { Component, Input, OnInit } from '@angular/core';

import { Router } from '@angular/router';
import { User } from './user';
import { UserService } from './user.service';


@Component({
    moduleId: module.id,
    selector: 'users',
    templateUrl: '/app/components/user-list.html',
    styleUrls: ['../../app/components/user.css'],
    providers: [UserService]
})


export class UsersComponent implements OnInit {
    users: User[];
    selectedUser: User;
    @Input() alias: string;
    @Input() email: string;
    @Input() password: string;

    constructor(private UserService: UserService, private router: Router) { }


    getUsers(): void {
        this.UserService.getUsers().then(Users => this.users = Users);
    }

    add(alias: string, email: string, password: string): void {
        email = email.trim();
        if (!email) { return; };
        this.UserService.create(alias, email, password)
            .then(User => {
                this.getUsers(); // refresh user list 
            });
    }

    delete(user: User): void {
        console.log('delete user ' + user.alias + ' ' + user.email + ' ' + user.userID);
        this.UserService
            .delete(user.userID)
            .then(() => {
                this.users = this.users.filter(h => h !== user );
                if (this.selectedUser === user ) {
                    this.selectedUser = null;
                }
            });
    }

    ngOnInit(): void {
        this.getUsers();
    }

    onSelect(User: User): void {
        this.selectedUser = User;
    }

    gotoDetail(): void {
        this.router.navigate(['/userdetail', this.selectedUser.userID]);
    }
}
