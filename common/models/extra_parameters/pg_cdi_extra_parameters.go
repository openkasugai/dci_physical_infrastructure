package extra_parameters

type PgCDIExtraParameters struct {
	CDIGuest     		string `json:"cdi_guest" validate:"required,min=1,max=128"`
	CDIUser     		string `json:"cdi_user" validate:"required,min=1,max=32"`
	CDIPassword  		string `json:"cdi_password" validate:"required,min=1,max=32"`
	CDIMgrGuestUser     string `json:"cdimgr_guest_user"`
	CDIMgrGuestPassword string `json:"cdimgr_guest_password"`
	CDIMgrHost 			string `json:"cdimgr_host"`
	CDIMgrHostUser      string `json:"cdimgr_host_user"`
	CDIMgrHostPassword  string `json:"cdimgr_host_password"`
	DirectorPassword    string `json:"director_password"`
}
