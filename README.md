# PHD2 Calibrate
PHD2 Calibrate is used to start the calibration process of [PHD2 Guiding](https://www.openphdguiding.org) via a simple command line executable.
This allows to trigger a calibration within scripts or imaging plans (e.g. the imaging plans provided by [APT - Astro Photography Tool](https://www.astrophotography.app)).

PHD2 Calibrate will automatically reset the existing calibration and start a new one by starting the image capturing, autoselecting an appropriate guide star and starting the calibration run.
After the calibration is finished or an error occurs the image capturing will be stopped automatically.

The default timeout limit of PHD2 Calibrate is set to 300 seconds - if the calibration does not finish within this timeframe the calibration is stopped. You can control the timeout limit via the command line flag "-t".

Example: "phd2calibrate -t=600" increases the timeout to 600 seconds

Prerequesites:
- PHD2 Guiding must be running
- Camera and mount must be connected
- PHD2 Server must be running (check option "Tools" menu of PHD2 Guiding)

A Microsoft Windows binary executable file of PHD2 Calibrate can be downloaded in the [Releases Section](https://github.com/ngaertner/phd2calibrate/releases).
You can also build your own binary using the latest source for your OS with [golang](https://golang.org).
