import { Pipe, PipeTransform } from '@angular/core';
import  * as lic from './partialLicense';

/*
 * Return only the rights of a partial license (or undefined)
 * Takes partialLicense as a string argument
 * Usage:
 *   partialLicense | filterRights
 */
@Pipe({name: 'FilterRights'})
export class FilterRights implements PipeTransform {

  transform(partialLicense: string): lic.UserRights | undefined  {
      let r: lic.UserRights = new lic.UserRights;
        let obj: any;
        obj = JSON.parse(partialLicense, function (key, value): any  {
          if (typeof value === 'string') {
            let a = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2}(?:\.\d*)?)Z$/.exec(value);
            if (a) {
              return new Date(Date.UTC(+a[1], +a[2] - 1, +a[3], +a[4], +a[5], +a[6]));
            }
          }
          return value;
        });
        if ( obj.rights ) {
            console.log(obj.rights);
            r = obj.rights;
            return r;
        }
        return undefined;
  }
}

@Pipe({name: 'ShowRights'})
export class ShowRights implements PipeTransform {

  transform(partialLicense: string): string  {
      let r: lic.UserRights = new lic.UserRights;
        let obj: any;
        obj = JSON.parse(partialLicense, function (key, value): any  {
          if (typeof value === 'string') {
            let a = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2}(?:\.\d*)?)Z$/.exec(value);
            if (a) {
              return new Date(Date.UTC(+a[1], +a[2] - 1, +a[3], +a[4], +a[5], +a[6]));
            }
          }
          return value;
        });
        if ( obj.rights ) {
            console.log(obj.rights);
            r = obj.rights;
            let s: string = '';
            if ( r.copy >0 ) {
                s = 'copy=' + r.copy + ', print=' + r.print + ' ';
            }
            return s + 'available from ' + r.start.toLocaleString() + ' to ' + r.end.toLocaleString();
        }
        return '';
  }
}