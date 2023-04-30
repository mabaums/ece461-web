import { Component, OnDestroy } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar'
import { Router, ActivatedRoute } from '@angular/router';
@Component({
  selector: 'app-nav-bar',
  templateUrl: './nav-bar.component.html',
  styleUrls: ['./nav-bar.component.scss']
})
export class NavBarComponent {
  searchText?: string = "";
  constructor(private _snackBar: MatSnackBar, private router: Router) {}

  search() {
    this.router.navigate(['/search/advanced'], {queryParams:{name: this.searchText} });
  }

  searchTextChange(event: any) {
    this.searchText = event.target.value;
  }

  openSnackBar(message: string, action: string) {
    this._snackBar.open(message, action);
  }
}
