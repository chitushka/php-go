<?php
interface ExistsAutoloadContract {
    public function run();
}

class ExistsAutoloaded {
    public $name = 'Ada';
    private $secret = 'hidden';

    public function run() {}
}
