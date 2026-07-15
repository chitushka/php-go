<?php
interface InspectContract {
    public function run();
}

class InspectBase {
    private $hidden = 1;

    protected function hiddenMethod() {
    }
}

class InspectTarget extends InspectBase implements InspectContract {
    public $name = 'Ada';

    private function secret() {
    }

    public function run() {
        return get_class();
    }
}
