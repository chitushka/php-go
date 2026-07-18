<?php
class WakeAuto {
    public $ready = 'no';

    public function __wakeup() {
        $this->ready = 'yes';
    }
}
