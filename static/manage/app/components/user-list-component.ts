import {Component} from '@angular/core';
import { OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { User } from './user';
import { UserService } from './user.service';


@Component({
    moduleId: module.id,
    selector: 'users',
    templateUrl: '/app/components/users.html',
    // styleUrls: ['user.css'],
    providers: [UserService]
})


export class UsersComponent implements OnInit {
    users: User[];
    selectedUser: User;

    constructor(private UserService: UserService, private router: Router) { }


    getUseres(): void {
        this.UserService.getUsers().then(Users => this.users = Users);
    }

    add(alias: string, email: string, password: string): void {
        email = email.trim();
        if (!email) { return; };
        this.UserService.create(alias, email, password)
            .then(User => {
                this.users.push(User);
                this.selectedUser = null;
            });
    }

    delete(User: User): void {
        this.UserService
            .delete(User.id)
            .then(() => {
                this.users = this.users.filter(h => h !== User );
                if (this.selectedUser === User ) {
                    this.selectedUser = null;
                }
            });
    }

    ngOnInit(): void {
        this.getUseres();
    }

    onSelect(User: User): void {
        this.selectedUser = User;
    }

    gotoDetail(): void {
        this.router.navigate(['/detail', this.selectedUser.id]);
    }
}
