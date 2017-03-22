import { Pipe, PipeTransform } from '@angular/core';

@Pipe({name: 'test'})
export class Sort implements PipeTransform {
  transform(value: {}[], filter: string, reverse: boolean): any {
    var values: {a:string, b:string};

    var getValue = (a:any, b:any, filter:string):{a:string, b:string} =>
    {
      var newA:string;
      var newB:string;

      var newFilter: string[] = filter.split(".")
      for (var i = 0; i <= newFilter.length-1; i++)
      {
        a = a[newFilter[i]];
        b = b[newFilter[i]];
      }

      return {
        a,
        b
      }
    }

    if(!reverse){
      value.sort(function (a:{}, b:{}) {
        values = getValue(a,b,filter);
        if (values.a == values.b) return 0;
        else if (values.a > values.b || values.a == null) return 1;
        else if (values.a < values.b || values.b == null) return -1;
      });
    } else {

      value.sort(function (a:{}, b:{}) {
        values = getValue(a,b,filter);
        if (values.a == values.b) return 0;
        else if (values.a < values.b || values.a == null) return 1;
        else  if (values.a > values.b || values.b == null) return -1;
      });
      
    }
    return value;

    
  }

  
  
}
